-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Votes` (
  `PostId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `Type` varchar(26) NOT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`UserId`,`Type`,`PostId`),
  KEY `idx_votes_user_id_type_create_at` (`UserId`,`Type`,`CreateAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Votes`;
