package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	NO_PENDING_REVIEWS = "no pending reviews for today"
)



// LearningStat 对应 DB 记录，包含 SM-2 算法所需核心数据
type LearningStat struct {
	WordID         int64
	EF             float64   // Easiness Factor (1.3-2.5)，初始值 2.5
	Repetitions    int       // 复习次数（答对后累加，答错归零）
	Interval       int       // 当前间隔天数
	NextReviewTime time.Time // 下次复习时间
}

// ReviewItem 复习队列项
type ReviewItem struct {
	Stat        *LearningStat
	WordDesc    *wordDesc // 复习时需要展示的单词详情
	ScheduledAt int       // 轮次 (1,2,3...)
}

const (
	REVIEWING = iota
	REVIEW_OVER
)

// ReviewSession 会话状态
type ReviewSession struct {
	Status      int
	UserId      string
	BookID      string
	BookName    string
	ReviewQueue []*ReviewItem
	CurrentIdx  int
}

// 用户登录成功后，将book_id和book_name保存起来
// StartReview：开始复习
// userNoteWords[uid][book_name] = []word_id

func StartReview(sid string, bookName string) map[string]bool {
	uid, err := redisClient.GetUserSession(sid)
	if err != nil {
		log.Fatal(err)
	}
	bookId, err := redisClient.GetUserBookId(uid, bookName)
	review, err := GetReview(uid, bookId, 10)
	if err != nil {
		log.Fatal(err)
	}
	for thisTurn := review.GetNext(); thisTurn != nil; thisTurn = review.GetNext() {
		fmt.Println(thisTurn.WordDesc.Word)
		fmt.Println("0.认识 1.不认识 2.猜一猜")
		choose := 0
		_, err := fmt.Scan(&choose)
		for err != nil {
			fmt.Println("输入错误，请重试")
			_, err = fmt.Scan(&choose)
		}
		switch choose {
		case 0:
			review.SubmitAnswer(thisTurn, true)
			thisTurn.WordDesc.show()
		case 1:
			review.SubmitAnswer(thisTurn, false)
			thisTurn.WordDesc.show()
		case 2:
			thisTurn.WordDesc.showExample()
			fmt.Scan(&choose)
			thisTurn.WordDesc.show()
		}
	}
	err = review.saveProgress()
	if err != nil {
		fmt.Println(err)
	}
	words := map[string]bool{}
	for _, item := range review.ReviewQueue {
		words[item.WordDesc.Word] = true
	}
	return words
}
func GetReview(uid, bookName string, limit int) (*ReviewSession, error) {
	// 1. 获取需要复习的记录 (包含算法属性 + 单词详情)
	stats, err := fetchReviewStats(uid, bookName, limit)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("no pending reviews for today")
	}
	for _, item := range stats {
		log.Println(item.WordDesc.Word)
	}
	// 2. 生成多轮次队列
	fmt.Println("生成多轮次队列")
	queue := generateQueue(stats)
	for _, item := range queue {
		log.Println(item.WordDesc.Word)
	}
	return &ReviewSession{
		UserId:      uid,
		BookName:    bookName,
		ReviewQueue: queue,
		CurrentIdx:  0,
	}, nil
}

// GetNext 获取下一题
func (s *ReviewSession) GetNext() *ReviewItem {
	if s.CurrentIdx >= len(s.ReviewQueue) {
		return nil
	}
	item := s.ReviewQueue[s.CurrentIdx]
	s.CurrentIdx++
	return item
}

// SubmitAnswer 提交并更新进度（SM-2 算法）
// isCorrect: true=认识 -> q=4, false=不认识 -> q=1
func (s *ReviewSession) SubmitAnswer(item *ReviewItem, isCorrect bool) {
	quality := 1 // 不认识 -> q=1
	if isCorrect {
		quality = 4 // 认识 -> q=4
	}
	updateSM2(item.Stat, quality)
	if s.ReviewQueue[len(s.ReviewQueue)-1] == item {
		s.Status = REVIEW_OVER
	}
}

// updateSM2 实现标准 SM-2 间隔重复算法
// quality: 0-5，本项目简化为 1=不认识, 4=认识
func updateSM2(s *LearningStat, quality int) {
	// 1. 更新 EF (Easiness Factor)
	// 公式: EF' = EF + (0.1 - (5-q) * (0.08 + (5-q) * 0.02))
	q := float64(quality)
	s.EF = s.EF + (0.1 - (5-q)*(0.08+(5-q)*0.02))
	if s.EF < 1.3 {
		s.EF = 1.3
	}
	if s.EF > 2.5 {
		s.EF = 2.5
	}

	// 2. 计算间隔
	if quality < 3 {
		// 答错：重置到第一次复习
		s.Repetitions = 0
		s.Interval = 1
	} else {
		// 答对：根据复习次数计算间隔
		s.Repetitions++
		switch s.Repetitions {
		case 1:
			s.Interval = 1
		case 2:
			s.Interval = 6
		default:
			// I(n) = I(n-1) * EF
			s.Interval = int(math.Round(float64(s.Interval) * s.EF))
		}
	}

	// 3. 设置下次复习时间（加随机抖动 ±10% 防止复习堆积）
	jitter := 0.9 + rand.Float64()*0.2
	days := float64(s.Interval) * jitter
	s.NextReviewTime = time.Now().Add(time.Duration(days*24) * time.Hour)
}

// generateQueue 生成复习队列：根据 EF 和 Repetitions 决定每个单词在 session 内出现次数，
// 并使用轮次分桶 + 随机打乱确保同一单词交错出现
func generateQueue(stats []*ReviewItem) []*ReviewItem {
	const numBuckets = 6 // 分成 6 轮
	buckets := make([][]*ReviewItem, numBuckets)
	for i := range buckets {
		buckets[i] = make([]*ReviewItem, 0)
	}

	for _, item := range stats {
		// 出现次数逻辑：新词或困难词多次出现
		times := 1
		if item.Stat.Repetitions == 0 {
			times = 3 // 新词出现 3 次
		} else if item.Stat.EF < 1.8 {
			times = 2 // 困难词（EF 低）出现 2 次
		}

		// 均匀分配到不同轮次，确保同一单词不相邻
		for i := 0; i < times; i++ {
			// 均匀分布到 0 ~ numBuckets-1 轮
			round := i * numBuckets / times
			newItem := *item // 浅拷贝结构体（Stat 指针共享，状态只更新一次）
			newItem.ScheduledAt = round + 1
			buckets[round] = append(buckets[round], &newItem)
		}
	}

	// 每轮内部随机打乱，然后按轮次顺序拼接
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var queue []*ReviewItem
	for _, bucket := range buckets {
		r.Shuffle(len(bucket), func(i, j int) {
			bucket[i], bucket[j] = bucket[j], bucket[i]
		})
		queue = append(queue, bucket...)
	}

	return queue
}

// fetchReviewStats 从 DB 拉取需要复习的单词（SM-2 字段）
func fetchReviewStats(uid, bookName string, limit int) ([]*ReviewItem, error) {
	// 查询 SM-2 所需字段：ef, repetitions, interval_days, next_review_time
	// 优先复习到期的(next <= now)，其次按 EF 升序（困难词优先）
	query := `SELECT 
		lr.word_id, lr.ef, lr.repetitions, lr.interval_days, lr.next_review_time
	FROM learning_record lr
	JOIN vocabulary v ON lr.word_id = v.id
	WHERE lr.user_id = ? AND lr.book_name = ? 
		AND (lr.next_review_time <= NOW() OR lr.next_review_time IS NULL)
	ORDER BY lr.ef ASC, lr.next_review_time ASC
	LIMIT ?`

	rows, err := db.Query(query, uid, bookName, limit)
	if err != nil {
		return nil, errors.New("query learning_record error: " + err.Error())
	}
	defer rows.Close()

	var list []*ReviewItem
	var wordIDs []int64
	for rows.Next() {
		s := &LearningStat{EF: 2.5} // 默认 EF 为 2.5
		var ef sql.NullFloat64
		var repetitions, intervalDays sql.NullInt64
		var nextReviewTime sql.NullTime

		if err := rows.Scan(&s.WordID, &ef, &repetitions, &intervalDays, &nextReviewTime); err != nil {
			return nil, err
		}
		if ef.Valid {
			s.EF = ef.Float64
		}
		if repetitions.Valid {
			s.Repetitions = int(repetitions.Int64)
		}
		if intervalDays.Valid {
			s.Interval = int(intervalDays.Int64)
		} else {
			s.Interval = 1
		}
		if nextReviewTime.Valid {
			s.NextReviewTime = nextReviewTime.Time
		}
		wordIDs = append(wordIDs, s.WordID)
		list = append(list, &ReviewItem{
			Stat:     s,
			WordDesc: nil,
		})
	}

	if len(wordIDs) == 0 {
		return list, nil
	}

	wordMap, err := selectWordsByIds(wordIDs...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		wordDesc, ok := wordMap[item.Stat.WordID]
		if !ok {
			return nil, fmt.Errorf("word not found for id: %d", item.Stat.WordID)
		}
		item.WordDesc = wordDesc
	}
	return list, nil
}

// saveProgress 将复习结果写回 DB（SM-2 字段）
func (session *ReviewSession) saveProgress() error {
	if session.Status == REVIEWING {
		return errors.New("session is not over")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 去重：同一单词可能在队列中出现多次，只保存最后一次的状态
	seen := make(map[int64]bool)
	uid := session.UserId
	bookName := session.BookName
	for i := len(session.ReviewQueue) - 1; i >= 0; i-- {
		stat := session.ReviewQueue[i]
		s := stat.Stat
		if seen[s.WordID] {
			continue
		}
		seen[s.WordID] = true
		_, err = tx.Exec(
			`UPDATE learning_record 
			 SET ef=?, repetitions=?, interval_days=?, next_review_time=?, 
			     total_reviews=total_reviews+1, last_review_time=NOW() 
			 WHERE user_id=? AND book_name=? AND word_id=?`,
			s.EF, s.Repetitions, s.Interval, s.NextReviewTime, uid, bookName, s.WordID,
		)
		if err != nil {
			return err
		}
	}
	// 更新用户连续打卡天数
	_, err = tx.Exec("UPDATE user SET streak=streak+1 WHERE id = ?", session.UserId)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
