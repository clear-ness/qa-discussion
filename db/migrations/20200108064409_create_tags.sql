-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `Tags` (
  `Content` varchar(64) NOT NULL,
  `PostCount` int(11) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `UpdateAt` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`Content`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Tags`;
