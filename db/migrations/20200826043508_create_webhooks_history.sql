-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `WebhooksHistory` (
  `Id` varchar(26) NOT NULL,
  `WebhookId` varchar(26) NOT NULL,
  `PostId` varchar(26) DEFAULT NULL,
  `TeamId` varchar(26) DEFAULT NULL,
  `WebhookName` varchar(64) DEFAULT NULL,
  `URL` text,
  `ContentType` varchar(128) DEFAULT NULL,
  `RequestBody` text,
  `ResponseBody` text,
  `ResponseStatus` int(11) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `WebhooksHistory`;
