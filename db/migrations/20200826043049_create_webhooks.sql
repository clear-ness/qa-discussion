-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Webhooks` (
  `Id` varchar(26) NOT NULL,
  `Token` varchar(26) DEFAULT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `QuestionEvents` tinyint(1) DEFAULT NULL,
  `AnswerEvents` tinyint(1) DEFAULT NULL,
  `CommentEvents` tinyint(1) DEFAULT NULL,
  `URLs` text,
  `Name` varchar(64) DEFAULT NULL,
  `Description` varchar(255) DEFAULT NULL,
  `ContentType` varchar(128) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `idx_webhook_team_id` (`TeamId`),
  KEY `idx_webhook_create_at` (`CreateAt`),
  KEY `idx_webhook_update_at` (`UpdateAt`),
  KEY `idx_webhook_delete_at` (`DeleteAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Webhooks`;
