-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Groups` (
  `Id` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `Name` varchar(64) DEFAULT NULL,
  `Description` varchar(255) DEFAULT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  UNIQUE KEY `Name` (`Name`,`TeamId`),
  KEY `idx_groups_team_id` (`TeamId`),
  KEY `idx_groups_name` (`Name`),
  KEY `idx_groups_update_at` (`UpdateAt`),
  KEY `idx_groups_create_at` (`CreateAt`),
  KEY `idx_groups_delete_at` (`DeleteAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Groups`;
