package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
	"log"
)

// LearningStat 对应 DB 记录，包含算法所需核心数据
type LearningStat struct {
	WordID             int64
	Familiarity        int       // 0-5
	ConsecutiveCorrect int       // 连续正确次数
	NextReviewTime     time.Time // 下次复习时间
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
	UserId    	string
	BookID      string
	BookName    string
	ReviewQueue []*ReviewItem
	CurrentIdx  int
}

// 用户登录成功后，将book_id和book_name保存起来
// StartReview：开始复习
// userNoteWords[uid][book_name] = []word_id

func StartReview(sid string){
	uid := userSession[sid]
	review, err := GetReview(uid, userBookToId[uid+"_我的生词本"], 10)
	if err != nil {
		log.Fatal(err)
	}
	for thisTurn := review.GetNext() ; thisTurn != nil; thisTurn = review.GetNext(){
		fmt.Println(thisTurn.WordDesc.Word)
		fmt.Println("0.认识 1.不认识 2.猜一猜")
		choose := 0
		_, err := fmt.Scan(&choose)
		for err != nil{
			fmt.Println("输入错误，请重试")
			_, err = fmt.Scan(&choose)
		}
		switch choose{
			case 0:
				review.SubmitAnswer(thisTurn, true)
				thisTurn.WordDesc.showWord()
			case 1:
				review.SubmitAnswer(thisTurn, false)
				thisTurn.WordDesc.showWord()
			case 2:
				thisTurn.WordDesc.showExample()
				fmt.Scan(&choose)
				thisTurn.WordDesc.showWord()
		}
	}
	err = review.saveProgress()
	if err != nil{
		fmt.Println(err)
	}
}
func GetReview(uid, bookID string, limit int) (*ReviewSession, error) {
	// 1. 获取需要复习的记录 (包含算法属性 + 单词详情)
	stats, err := fetchReviewStats(uid, bookID, limit)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("no pending reviews for today")
	}

	// 2. 生成多轮次队列
	queue := generateQueue(stats)

	return &ReviewSession{
		UserId:   uid, 
		BookID:      bookID,
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

// SubmitAnswer 提交并更新进度
// 简化：直接在 ReviewSession 里处理逻辑，不用每次都去 DB 查一遍 stats
func (s *ReviewSession) SubmitAnswer(item *ReviewItem, isCorrect bool) {
	stat := item.Stat
	updateFamiAndNextReview(stat, isCorrect)
	if s.ReviewQueue[len(s.ReviewQueue)-1] == item {
		s.Status = REVIEW_OVER
	}
}

// ---------------------------------------------------------------------------
// 算法逻辑 (SM-2 简化)
// ---------------------------------------------------------------------------

func updateFamiAndNextReview(s *LearningStat, isCorrect bool) {
	if isCorrect {
		s.ConsecutiveCorrect++
		if s.Familiarity < 5 {
			s.Familiarity++
		}
	} else {
		s.ConsecutiveCorrect = 0
		if s.Familiarity > 0 {
			s.Familiarity -= 2 // 答错扣分狠一点
			if s.Familiarity < 0 {
				s.Familiarity = 0
			}
		}
	}

	// 计算间隔 (Days)
	intervals := []float64{0.5, 1, 3, 7, 15, 30}
	days := intervals[s.Familiarity]

	// 加上微小的随机抖动 (±10%) 防止复习堆积
	days *= (0.9 + rand.Float64()*0.2)

	s.NextReviewTime = time.Now().Add(time.Duration(days*24) * time.Hour)
}

func generateQueue(stats []*ReviewItem) []*ReviewItem {
	var queue []*ReviewItem
	for _, item := range stats {
		// 出现次数逻辑：Familiarity越低，出现次数越多 (Max 3, Min 1)
		times := 1
		if item.Stat.Familiarity <= 1 {
			times = 3 - item.Stat.Familiarity // 0->3次, 1->2次
		}

		// 分配轮次
		for i := 0; i < times; i++ {
			// 映射到 [1..6] 轮
			round := 1
			if times > 1 {
				// 线性分布：第0次->1轮，第(times-1)次->6轮
				round = 1 + int(math.Round(float64(i)*5.0/float64(times-1)))
			}

			// 深拷贝 item 的一部分引用，或者直接复用指针 (注意 item.Stat 指针共享状态)
			newItem := *item // 浅拷贝结构体
			newItem.ScheduledAt = round
			queue = append(queue, &newItem)
		}
	}

	// 对队列按 (Round, Random) 排序
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sort.Slice(queue, func(i, j int) bool {
		if queue[i].ScheduledAt != queue[j].ScheduledAt {
			return queue[i].ScheduledAt < queue[j].ScheduledAt
		}
		// 同轮次内随机
		return r.Intn(2) == 0
	})

	return queue
}

// ---------------------------------------------------------------------------
// DB 操作简化
// ---------------------------------------------------------------------------

func fetchReviewStats(uid, bookID string, limit int) ([]*ReviewItem, error) {
	// JOIN 查询：一次性拿出 复习进度 + 单词基本信息
	// 优先复习到期的(next <= now)，其次是新词(next_review_time IS NULL 或 total_reviews=0)
	query := `SELECT 
		lr.word_id, lr.familiarity, lr.consecutive_correct, lr.next_review_time
	FROM learning_record lr
	JOIN vocabulary v ON lr.word_id = v.id
	WHERE lr.user_id = ? AND lr.book_id = ? 
		AND (lr.next_review_time <= NOW() OR lr.next_review_time IS NULL)
	ORDER BY lr.familiarity ASC, lr.next_review_time ASC
	LIMIT ?
`
	// 调试：打印查询参数
	// fmt.Printf("DEBUG fetchReviewStats: uid=%s, bookID=%s, limit=%d\n", uid, bookID, limit)
	rows, err := db.Query(query, uid, bookID, limit)
	if err != nil {
		return nil, errors.New("query learning_record and vocabulary error " + err.Error())
	}
	defer rows.Close()

	var list []*ReviewItem
	for rows.Next() {
		s := &LearningStat{}

		// 注意 Scan 顺序
		// 如果 DSN 中包含 parseTime=true，可以直接扫描到 time.Time
		// 否则需要先扫描到 string 再解析
		var nextReviewTime sql.NullTime // 使用 NullTime 处理可能的 NULL 值

		if err := rows.Scan(&s.WordID, &s.Familiarity, &s.ConsecutiveCorrect, &nextReviewTime); err != nil {
			return nil, err
		}

		// 处理 NULL 值：如果为 NULL，设置为零值或当前时间
		if nextReviewTime.Valid {
			s.NextReviewTime = nextReviewTime.Time
		} else {
			s.NextReviewTime = time.Time{} // 零值，表示未设置
		}

		list = append(list, &ReviewItem{
			Stat:     s,
			WordDesc: wordsPool[s.WordID],
		})
	}
	return list, nil
}

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

	stats := session.ReviewQueue
	uid := session.UserId
	bookId := session.BookID
	for _, stat := range stats {
		s := stat.Stat
		_, err = tx.Exec(
			"UPDATE learning_record SET familiarity=?, consecutive_correct=?, next_review_time=?, total_reviews=total_reviews+1, last_review_time=NOW() WHERE user_id=? AND book_id=? AND word_id=?",
			s.Familiarity, s.ConsecutiveCorrect, s.NextReviewTime, uid, bookId, s.WordID,
		)
		if err != nil {
			return err
		}
	}
	// 提交事务
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
