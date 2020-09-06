-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `OAuthAuthorizedApps` (
  `UserId` varchar(26) NOT NULL,
  `ClientId` varchar(26) NOT NULL,
  `Scope` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`UserId`,`ClientId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `OAuthAuthorizedApps`;
