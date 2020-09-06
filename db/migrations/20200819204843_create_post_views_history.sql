-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `PostViewsHistory` (
  `Id` varchar(26) NOT NULL,
  `PostId` varchar(26) NOT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  `IpAddress` varchar(64) DEFAULT NULL,
  `ViewsCount` int(11) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `PostViewsHistory`;
