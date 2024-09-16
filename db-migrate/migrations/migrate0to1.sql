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

-- Create "new_hosts" table
CREATE TABLE `new_hosts` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `host_name` text NOT NULL,
  `sequence_id` integer NOT NULL,
  `eth` text NULL,
  `mac` text NOT NULL,
  `ip` text NULL,
  `boot_mode` text NOT NULL DEFAULT 'bios',
  `state` integer NULL,
  `restore_state` integer NULL,
  `cluster_id` integer NOT NULL,
  `host_policy_id` integer NULL,
  CONSTRAINT `fk_host_policies_hosts` FOREIGN KEY (`host_policy_id`) REFERENCES `host_policies` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_clusters_hosts` FOREIGN KEY (`cluster_id`) REFERENCES `clusters` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "hosts" to new temporary table "new_hosts"
INSERT INTO `new_hosts` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `host_name`, `sequence_id`, `eth`, `mac`, `ip`, `state`, `cluster_id`, `host_policy_id`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `name`, `host_name`, `sequence_id`, `eth`, `mac`, `ip`, `state`, `cluster_id`, `host_policy_id` FROM `hosts`;
-- Drop "hosts" table after copying rows
DROP TABLE `hosts`;
-- Rename temporary table "new_hosts" to "hosts"
ALTER TABLE `new_hosts` RENAME TO `hosts`;
-- Create index "hosts_mac" to table: "hosts"
CREATE UNIQUE INDEX `hosts_mac` ON `hosts` (`mac`);
-- Create index "hosts_name" to table: "hosts"
CREATE UNIQUE INDEX `hosts_name` ON `hosts` (`name`);
-- Create index "hosts_host_name" to table: "hosts"
CREATE UNIQUE INDEX `hosts_host_name` ON `hosts` (`host_name`);
-- Create index "idx_cluster_seq" to table: "hosts"
CREATE UNIQUE INDEX `idx_cluster_seq` ON `hosts` (`sequence_id`, `cluster_id`);

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

-- Create "new_distro_images" table
CREATE TABLE `new_distro_images` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `image_id` text NOT NULL,
  `type` text NOT NULL,
  `name` text NOT NULL,
  `kernel` text NULL,
  `initrd` text NULL,
  `breed` text NULL,
  `local_boot` numeric NULL,
  `bios_boot` numeric NOT NULL DEFAULT false,
  `uefi_boot` numeric NOT NULL DEFAULT false
);
-- Copy rows from old table "distro_images" to new temporary table "new_distro_images"
INSERT INTO `new_distro_images` (`id`, `created_at`, `updated_at`, `deleted_at`, `image_id`, `type`, `name`, `kernel`, `initrd`, `breed`, `local_boot`, `bios_boot`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `image_id`, `type`, `name`, `kernel`, `initrd`, `breed`, `local_boot`, 1 FROM `distro_images`;
-- Drop "distro_images" table after copying rows
DROP TABLE `distro_images`;
-- Rename temporary table "new_distro_images" to "distro_images"
ALTER TABLE `new_distro_images` RENAME TO `distro_images`;
-- Create index "distro_images_image_id" to table: "distro_images"
CREATE UNIQUE INDEX `distro_images_image_id` ON `distro_images` (`image_id`);
-- Create index "distro_images_name" to table: "distro_images"
CREATE UNIQUE INDEX `distro_images_name` ON `distro_images` (`name`);

-- Create "new_groups_users" table
CREATE TABLE `new_groups_users` (
  `user_id` integer NULL,
  `group_id` integer NULL,
  PRIMARY KEY (`user_id`, `group_id`),
  CONSTRAINT `fk_groups_users_group` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_groups_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "groups_users" to new temporary table "new_groups_users"
INSERT INTO `new_groups_users` (`user_id`, `group_id`) SELECT `user_id`, `group_id` FROM `groups_users`;
-- Drop "groups_users" table after copying rows
DROP TABLE `groups_users`;
-- Rename temporary table "new_groups_users" to "groups_users"
ALTER TABLE `new_groups_users` RENAME TO `groups_users`;

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
