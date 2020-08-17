-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Teams` (
  `Id` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `Name` varchar(64) DEFAULT NULL,
  `Description` varchar(255) DEFAULT NULL,
  `Email` varchar(128) DEFAULT NULL,
  `AllowedDomains` text,
  `InviteId` varchar(32) DEFAULT NULL,
  `LastPictureUpdate` bigint(20) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  UNIQUE KEY `Name` (`Name`),
  KEY `idx_teams_name` (`Name`),
  KEY `idx_teams_invite_id` (`InviteId`),
  KEY `idx_teams_update_at` (`UpdateAt`),
  KEY `idx_teams_create_at` (`CreateAt`),
  KEY `idx_teams_delete_at` (`DeleteAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Teams`;
