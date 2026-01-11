package main

import (
    _"math"
    "sort"
    "time"
	"fmt"
	// "rand"
)

// WordLearning 单词学习记录
type WordLearning struct {
    WordID             int64
    Word               string
    Familiarity        int       // 熟悉度 0-5
    ConsecutiveCorrect int       // 连续正确次数
    TotalReviews       int       // 总复习次数
    CorrectCount       int       // 正确次数
    WrongCount         int       // 错误次数
    LastReviewTime     time.Time // 上次复习时间
    NextReviewTime     time.Time // 下次应复习时间
    TodayReviews       int       // 今日复习次数
    TodayCorrect       int       // 今日正确次数
}

// ReviewSession 复习会话
type ReviewSession struct {
    UserID      string
    BookID      int64
    Words       []*WordLearning
    CurrentIdx  int
    ReviewQueue []*ReviewItem
}

// ReviewItem 复习队列项
type ReviewItem struct {
    WordLearning *WordLearning
    Priority     float64 // 优先级（越高越需要复习）
    ScheduledAt  int     // 计划在第几轮出现
}

// 基于 SM-2 算法的间隔重复系统
// 熟悉度对应的复习间隔（天数）
var intervalDays = map[int]float64{
    0: 0,      // 新词，当天复习
    1: 1,      // 1天后
    2: 3,      // 3天后
    3: 7,      // 7天后
    4: 15,     // 15天后
    5: 30,     // 30天后（已掌握）
}

// CalculateNextReview 计算下次复习时间
func CalculateNextReview(wl *WordLearning, isCorrect bool) time.Time {
    now := time.Now()
    
    if isCorrect {
        // 答对了，提升熟悉度
        if wl.Familiarity < 5 {
            wl.Familiarity++
        }
        wl.ConsecutiveCorrect++
    } else {
        // 答错了，降低熟悉度
        wl.Familiarity = max(0, wl.Familiarity-2)
        wl.ConsecutiveCorrect = 0
    }
    
    // 根据熟悉度计算间隔
    days := intervalDays[wl.Familiarity]
    
    // 根据正确率调整间隔
    if wl.TotalReviews > 0 {
        correctRate := float64(wl.CorrectCount) / float64(wl.TotalReviews)
        if correctRate < 0.6 {
            days *= 0.5 // 正确率低，缩短间隔
        } else if correctRate > 0.9 {
            days *= 1.2 // 正确率高，延长间隔
        }
    }
    
    return now.Add(time.Duration(days*24) * time.Hour)
}

// CalculatePriority 计算单词的复习优先级
func CalculatePriority(wl *WordLearning) float64 {
    now := time.Now()
    priority := 0.0
    
    // 1. 逾期时间因素（越逾期越优先）
	// 检查是否已超过截止时间
    if now.After(wl.NextReviewTime) {
        overdueDays := now.Sub(wl.NextReviewTime).Hours() / 24
        priority += overdueDays * 10
    }
    
    // 2. 熟悉度因素（越不熟悉越优先）
    priority += float64(5-wl.Familiarity) * 5
    
    // 3. 错误率因素
    if wl.TotalReviews > 0 {
        errorRate := float64(wl.WrongCount) / float64(wl.TotalReviews)
        priority += errorRate * 20
    }
    
    // 4. 今日复习次数（今日复习少的优先）
    priority += float64(3-min(wl.TodayReviews, 3)) * 3
    
    // 5. 新词优先
    if wl.TotalReviews == 0 {
        priority += 15
    }
    
    return priority
}

// SelectTodayWords 选择今日需要复习的单词
func SelectTodayWords(userID string, bookID int64, count int) ([]*WordLearning, error) {
    now := time.Now()
    // today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    
    // 从数据库查询需要复习的单词
    query := `
        SELECT word_id, familiarity, consecutive_correct, total_reviews,
               correct_count, wrong_count, last_review_time, next_review_time,
               today_reviews, today_correct
        FROM word_learning
        WHERE user_id = ? AND book_id = ?
          AND (next_review_time <= ? OR total_reviews = 0)
        ORDER BY next_review_time ASC
        LIMIT ?
    `
    
    rows, err := db.Query(query, userID, bookID, now, count*2)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var words []*WordLearning
    for rows.Next() {
        wl := &WordLearning{}
        err := rows.Scan(
            &wl.WordID, &wl.Familiarity, &wl.ConsecutiveCorrect,
            &wl.TotalReviews, &wl.CorrectCount, &wl.WrongCount,
            &wl.LastReviewTime, &wl.NextReviewTime,
            &wl.TodayReviews, &wl.TodayCorrect,
        )
        if err != nil {
            return nil, err
        }
        words = append(words, wl)
    }
    
    // 按优先级排序
    sort.Slice(words, func(i, j int) bool {
        return CalculatePriority(words[i]) > CalculatePriority(words[j])
    })
    
    // 取前 count 个
    if len(words) > count {
        words = words[:count]
    }
    
    return words, nil
}

// GenerateReviewQueue 生成复习队列
// 根据艾宾浩斯遗忘曲线，难词会在一轮复习中多次出现
func GenerateReviewQueue(words []*WordLearning) []*ReviewItem {
    var queue []*ReviewItem
    
    for _, wl := range words {
        // 根据熟悉度决定本轮出现次数
        appearances := CalculateAppearances(wl)
        
        for i := 0; i < appearances; i++ {
            item := &ReviewItem{
                WordLearning: wl,
                Priority:     CalculatePriority(wl),
                ScheduledAt:  calculateSchedulePosition(wl, i, appearances),
            }
            queue = append(queue, item)
        }
    }
    
    // 按计划位置排序，并加入随机性
    sort.Slice(queue, func(i, j int) bool {
        if queue[i].ScheduledAt != queue[j].ScheduledAt {
            return queue[i].ScheduledAt < queue[j].ScheduledAt
        }
        return queue[i].Priority > queue[j].Priority
    })
    
    // 打乱同一轮次内的顺序，避免机械记忆
    shuffleWithinRounds(queue)
    
    return queue
}

// CalculateAppearances 计算单词在本轮应出现的次数
func CalculateAppearances(wl *WordLearning) int {
    // 基础出现次数
    base := 1
    
    // 根据熟悉度调整
    switch wl.Familiarity {
    case 0:
        base = 4 // 新词出现4次
    case 1:
        base = 3 // 不熟悉出现3次
    case 2:
        base = 2 // 一般熟悉出现2次
    default:
        base = 1 // 熟悉的出现1次
    }
    
    // 根据今日答错情况调整
    if wl.TodayReviews > 0 {
        errorRate := float64(wl.TodayReviews-wl.TodayCorrect) / float64(wl.TodayReviews)
        if errorRate > 0.5 {
            base += 2
        } else if errorRate > 0 {
            base += 1
        }
    }
    
    return min(base, 5) // 最多出现5次
}

// calculateSchedulePosition 计算单词在队列中的位置
func calculateSchedulePosition(wl *WordLearning, appearanceIdx, totalAppearances int) int {
    // 将复习分成几轮，确保间隔出现
    // 例如：一个词出现3次，分别在第1轮、第3轮、第5轮出现
    roundInterval := 6 / totalAppearances
    return appearanceIdx*roundInterval + 1
}

// shuffleWithinRounds 在同一轮次内打乱顺序
func shuffleWithinRounds(queue []*ReviewItem) {
    // 按轮次分组
    rounds := make(map[int][]*ReviewItem)
    for _, item := range queue {
        rounds[item.ScheduledAt] = append(rounds[item.ScheduledAt], item)
    }
    
    // 打乱每一轮的顺序
    // for _, items := range rounds {
    //     rand.Shuffle(len(items), func(i, j int) {
    //         items[i], items[j] = items[j], items[i]
    //     })
    // }
}

// ProcessAnswer 处理用户答题结果
func ProcessAnswer(wl *WordLearning, isCorrect bool) error {
    now := time.Now()
    
    // 更新统计
    wl.TotalReviews++
    wl.TodayReviews++
    wl.LastReviewTime = now
    
    if isCorrect {
        wl.CorrectCount++
        wl.TodayCorrect++
        wl.ConsecutiveCorrect++
        
        // 连续答对，提升熟悉度
        if wl.ConsecutiveCorrect >= 2 && wl.Familiarity < 5 {
            wl.Familiarity++
        }
    } else {
        wl.WrongCount++
        wl.ConsecutiveCorrect = 0
        
        // 答错，降低熟悉度
        wl.Familiarity = max(0, wl.Familiarity-1)
    }
    
    // 计算下次复习时间
    wl.NextReviewTime = CalculateNextReview(wl, isCorrect)
    
    // 更新数据库
    return updateWordLearning(wl)
}

// updateWordLearning 更新数据库
func updateWordLearning(wl *WordLearning) error {
    query := `
        UPDATE word_learning SET
            familiarity = ?,
            consecutive_correct = ?,
            total_reviews = ?,
            correct_count = ?,
            wrong_count = ?,
            last_review_time = ?,
            next_review_time = ?,
            today_reviews = ?,
            today_correct = ?
        WHERE word_id = ?
    `
    _, err := db.Exec(query,
        wl.Familiarity, wl.ConsecutiveCorrect,
        wl.TotalReviews, wl.CorrectCount, wl.WrongCount,
        wl.LastReviewTime, wl.NextReviewTime,
        wl.TodayReviews, wl.TodayCorrect,
        wl.WordID,
    )
    return err
}


// StartReviewSession 开始复习会话
func StartReviewSession(userID string, bookID int64) (*ReviewSession, error) {
    // 1. 选择今日需要复习的20个单词
    words, err := SelectTodayWords(userID, bookID, 20)
    if err != nil {
        return nil, err
    }
    
    // 2. 生成复习队列
    queue := GenerateReviewQueue(words)
    
    session := &ReviewSession{
        UserID:      userID,
        BookID:      bookID,
        Words:       words,
        CurrentIdx:  0,
        ReviewQueue: queue,
    }
    
    return session, nil
}

// GetNextWord 获取下一个需要复习的单词
func (s *ReviewSession) GetNextWord() *WordLearning {
    if s.CurrentIdx >= len(s.ReviewQueue) {
        return nil // 复习完成
    }
    
    item := s.ReviewQueue[s.CurrentIdx]
    s.CurrentIdx++
    
    return item.WordLearning
}

// SubmitAnswer 提交答案
func (s *ReviewSession) SubmitAnswer(wordID int64, isCorrect bool) error {
    // 找到对应的单词
    for _, wl := range s.Words {
        if wl.WordID == wordID {
            return ProcessAnswer(wl, isCorrect)
        }
    }
    return fmt.Errorf("word not found in session")
}