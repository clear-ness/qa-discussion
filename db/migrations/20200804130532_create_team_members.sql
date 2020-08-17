-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `TeamMembers` (
  `TeamId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `Points` int(11) DEFAULT NULL,
  `DeleteAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`TeamId`,`UserId`),
  KEY `idx_teammembers_team_id` (`TeamId`),
  KEY `idx_teammembers_user_id` (`UserId`),
  KEY `idx_teammembers_delete_at` (`DeleteAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `TeamMembers`;
