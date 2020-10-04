-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Status` (
  `UserId` varchar(26) NOT NULL,
  `Status` varchar(32) DEFAULT NULL,
  `Manual` tinyint(1) DEFAULT NULL,
  `LastActivityAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`UserId`),
  KEY `idx_status_user_id` (`UserId`),
  KEY `idx_status_status` (`Status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Status`;
