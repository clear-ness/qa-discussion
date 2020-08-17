-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `InboxMessages` (
  `Id` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `Content` text,
  `UserId` varchar(26) DEFAULT NULL,
  `SenderId` varchar(26) DEFAULT NULL,
  `QuestionId` varchar(26) NOT NULL,
  `Title` text,
  `AnswerId` varchar(26) NOT NULL,
  `CommentId` varchar(26) NOT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `idx_inbox_messages_user_id_create_at` (`UserId`,`CreateAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `InboxMessages`;
