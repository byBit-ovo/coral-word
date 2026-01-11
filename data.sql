
use coral_word;
CREATE TABLE word_learning (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    word_id BIGINT NOT NULL,
    book_id BIGINT NOT NULL,
    
    familiarity INT DEFAULT 0,       
    consecutive_correct INT DEFAULT 0,   
    total_reviews INT DEFAULT 0,        
    correct_count INT DEFAULT 0,        
    wrong_count INT DEFAULT 0,          
    
  
    last_review_time DATETIME,          
    next_review_time DATETIME,           
    first_learn_time DATETIME,           
    
   
    today_reviews INT DEFAULT 0,        
    today_correct INT DEFAULT 0,         
    
    UNIQUE KEY unique_user_word (user_id, word_id),
    INDEX idx_next_review (user_id, book_id, next_review_time)
);