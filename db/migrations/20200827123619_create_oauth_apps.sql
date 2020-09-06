-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `OAuthApps` (
  `Id` varchar(26) NOT NULL,
  `UserId` varchar(26) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  `ClientSecret` varchar(128) DEFAULT NULL,
  `Name` varchar(64) DEFAULT NULL,
  `Description` text,
  `IconURL` text,
  `URLs` text,
  `Homepage` text,
  PRIMARY KEY (`Id`),
  KEY `idx_oauthapps_user_id` (`UserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `OAuthApps`;
