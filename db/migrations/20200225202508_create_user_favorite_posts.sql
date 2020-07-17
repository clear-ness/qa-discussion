-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `UserFavoritePosts` (
  `PostId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`PostId`,`UserId`),
  KEY `idx_user_favorite_posts_user_id_create_at` (`UserId`,`CreateAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `UserFavoritePosts`;
