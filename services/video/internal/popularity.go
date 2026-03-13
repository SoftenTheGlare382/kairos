package video

// Popularity 热度权重（点赞20% 评论40% 收藏30% 观看10%）
// 使用整数权重便于增量计算：2 + 4 + 3 + 1 = 10
const (
	PopularityWeightLike     int64 = 2 // 20%
	PopularityWeightComment  int64 = 4 // 40%
	PopularityWeightFavorite int64 = 3 // 30%
	PopularityWeightPlay     int64 = 1 // 10%
)
