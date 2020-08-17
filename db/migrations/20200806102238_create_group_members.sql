-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `GroupMembers` (
  `GroupId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `Type` varchar(26) DEFAULT NULL,
  PRIMARY KEY (`GroupId`,`UserId`),
  KEY `idx_groupmembers_group_id` (`GroupId`),
  KEY `idx_groupmembers_user_id` (`UserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `GroupMembers`;
