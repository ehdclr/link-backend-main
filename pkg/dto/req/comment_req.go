package req

type CommentRequest struct {
	PostID      uint   `json:"post_id" binding:"required"`
	IsAnonymous *bool  `json:"is_anonymous" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

type ReplyRequest struct {
	PostID      uint   `json:"post_id" binding:"required"`
	ParentID    uint   `json:"parent_id" binding:"required"`
	IsAnonymous *bool  `json:"is_anonymous" binding:"required"`
	Content     string `json:"content" binding:"required"`
}
