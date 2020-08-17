-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `TeamMemberHistory` (
  `TeamId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `JoinTime` bigint(20) NOT NULL,
  `LeaveTime` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`TeamId`,`UserId`,`JoinTime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `TeamMemberHistory`;
