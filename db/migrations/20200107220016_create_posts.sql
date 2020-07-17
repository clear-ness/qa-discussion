-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Posts` (
  `Id` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `ParentId` varchar(26) DEFAULT NULL,
  `RootId` varchar(26) DEFAULT NULL,
  `OriginalId` varchar(26) DEFAULT NULL,
  `BestId` varchar(26) DEFAULT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  `Title` text,
  `Content` text,
  `Tags` text,
  `Props` text,
  `UpVotes` int(11) DEFAULT NULL,
  `DownVotes` int(11) DEFAULT NULL,
  `Points` int(11) DEFAULT NULL,
  `AnswerCount` int(11) DEFAULT NULL,
  `FlagCount` int(11) DEFAULT NULL,
  `ProtectedAt` bigint(20) DEFAULT NULL,
  `LockedAt` bigint(20) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `EditAt` bigint(20) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `idx_posts_type_delete_at_create_at` (`Type`,`DeleteAt`,`CreateAt`),
  KEY `idx_posts_type_delete_at_update_at` (`Type`,`DeleteAt`,`UpdateAt`),
  KEY `idx_posts_type_delete_at_points` (`Type`,`DeleteAt`,`Points`),
  KEY `idx_posts_type_delete_at_answer_count` (`Type`,`DeleteAt`,`AnswerCount`),
  KEY `idx_posts_type_up_votes_delete_at_create_at` (`Type`,`UpVotes`,`DeleteAt`,`CreateAt`),
  KEY `idx_posts_parent_id_type_delete_at_create_at` (`ParentId`,`Type`,`DeleteAt`,`CreateAt`),
  KEY `idx_posts_parent_id_type_delete_at_update_at` (`ParentId`,`Type`,`DeleteAt`,`UpdateAt`),
  KEY `idx_posts_parent_id_type_delete_at_points` (`ParentId`,`Type`,`DeleteAt`,`Points`),
  KEY `idx_posts_root_id_type_delete_at_create_at` (`RootId`,`Type`,`DeleteAt`,`CreateAt`),
  KEY `idx_posts_user_id_type_delete_at_create_at` (`UserId`,`Type`,`DeleteAt`,`CreateAt`),
  FULLTEXT KEY `idx_posts_title_txt` (`Title`),
  FULLTEXT KEY `idx_posts_content_txt` (`Content`),
  FULLTEXT KEY `idx_posts_tags_txt` (`Tags`),
  FULLTEXT KEY `idx_posts_title_tags_content_txt` (`Title`,`Tags`,`Content`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Posts`;
