-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Collections` (
  `Id` varchar(26) NOT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `Title` text,
  `Description` varchar(255) DEFAULT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `idx_collections_team_id` (`TeamId`),
  KEY `idx_collections_create_at` (`CreateAt`),
  KEY `idx_collections_update_at` (`UpdateAt`),
  KEY `idx_collections_delete_at` (`DeleteAt`),
  FULLTEXT KEY `idx_collections_title_txt` (`Title`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Collections`;
