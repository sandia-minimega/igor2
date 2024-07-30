-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;

-- Create "groups_owners" table
CREATE TABLE `groups_owners` (
  `group_id` integer NULL,
  `user_id` integer NULL,
  PRIMARY KEY (`group_id`, `user_id`),
  CONSTRAINT `fk_groups_owners_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_groups_owners_group` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);

-- Copy the data from groups into the new lookup table
-- In this migration phase there is only a single owner for each group and permissions don't need updating
INSERT INTO groups_owners(group_id, user_id) SELECT id, owner_id FROM groups;

-- Create "new_maintenanceres_hosts" table
CREATE TABLE `new_maintenanceres_hosts` (
  `maintenance_res_id` integer NULL,
  `host_id` integer NULL,
  PRIMARY KEY (`maintenance_res_id`, `host_id`),
  CONSTRAINT `fk_maintenanceres_hosts_host` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_maintenanceres_hosts_maintenance_res` FOREIGN KEY (`maintenance_res_id`) REFERENCES `maintenance_res` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "maintenanceres_hosts" to new temporary table "new_maintenanceres_hosts"
INSERT INTO `new_maintenanceres_hosts` (`maintenance_res_id`, `host_id`) SELECT `maintenance_res_id`, `host_id` FROM `maintenanceres_hosts`;
-- Drop "maintenanceres_hosts" table after copying rows
DROP TABLE `maintenanceres_hosts`;
-- Rename temporary table "new_maintenanceres_hosts" to "maintenanceres_hosts"
ALTER TABLE `new_maintenanceres_hosts` RENAME TO `maintenanceres_hosts`;
-- Create "new_groups" table
CREATE TABLE `new_groups` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `is_user_private` numeric NULL,
  `is_ldap` numeric NULL DEFAULT false
);
-- Copy rows from old table "groups" to new temporary table "new_groups"
INSERT INTO `new_groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `is_user_private`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `is_user_private` FROM `groups`;
-- Drop "groups" table after copying rows
DROP TABLE `groups`;
-- Rename temporary table "new_groups" to "groups"
ALTER TABLE `new_groups` RENAME TO `groups`;
-- Create index "groups_name" to table: "groups"
CREATE UNIQUE INDEX `groups_name` ON `groups` (`name`);
-- Create "new_reservations_hosts" table
CREATE TABLE `new_reservations_hosts` (
  `host_id` integer NULL,
  `reservation_id` integer NULL,
  PRIMARY KEY (`host_id`, `reservation_id`),
  CONSTRAINT `fk_reservations_hosts_reservation` FOREIGN KEY (`reservation_id`) REFERENCES `reservations` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_reservations_hosts_host` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "reservations_hosts" to new temporary table "new_reservations_hosts"
INSERT INTO `new_reservations_hosts` (`host_id`, `reservation_id`) SELECT `host_id`, `reservation_id` FROM `reservations_hosts`;
-- Drop "reservations_hosts" table after copying rows
DROP TABLE `reservations_hosts`;
-- Rename temporary table "new_reservations_hosts" to "reservations_hosts"
ALTER TABLE `new_reservations_hosts` RENAME TO `reservations_hosts`;

CREATE TABLE `new_host_policies` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `max_res_time` integer NULL,
  `notavailable` text NULL
);

INSERT INTO `new_host_policies` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `max_res_time`, `notavailable`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `name`, `max_res_time`, `notavailable` FROM `host_policies`;
DROP TABLE `host_policies`;
ALTER TABLE `new_host_policies` RENAME TO `host_policies`;
CREATE UNIQUE INDEX `host_policies_name` ON `host_policies` (`name`);

-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
