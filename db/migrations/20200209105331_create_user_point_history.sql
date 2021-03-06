-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `UserPointHistory` (
  `Id` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `Type` varchar(26) DEFAULT NULL,
  `PostId` varchar(26) DEFAULT NULL,
  `PostType` varchar(26) DEFAULT NULL,
  `Tags` text,
  `Points` int(11) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `UserPointHistory`;
