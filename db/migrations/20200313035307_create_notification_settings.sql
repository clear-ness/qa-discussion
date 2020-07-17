-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `NotificationSettings` (
  `Id` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `InboxInterval` varchar(26) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `idx_notification_settings_user_id_inbox_interval` (`UserId`, `InboxInterval`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `NotificationSettings`;
