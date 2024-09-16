-- Create "maintenance_res" table
CREATE TABLE `maintenance_res` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `reservation_name` text NULL,
  `maintenance_end_time` datetime NULL
);
-- Create "clusters" table
CREATE TABLE `clusters` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `prefix` text NOT NULL,
  `display_height` integer NULL,
  `display_width` integer NULL,
  `motd` text NOT NULL,
  `motd_urgent` numeric NOT NULL
);
-- Create index "clusters_name" to table: "clusters"
CREATE UNIQUE INDEX `clusters_name` ON `clusters` (`name`);
-- Create index "clusters_prefix" to table: "clusters"
CREATE UNIQUE INDEX `clusters_prefix` ON `clusters` (`prefix`);
-- Create "host_policies" table
CREATE TABLE `host_policies` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `max_res_time` integer NULL,
  `notavailable` text NULL
);
-- Create index "host_policies_name" to table: "host_policies"
CREATE UNIQUE INDEX `host_policies_name` ON `host_policies` (`name`);
-- Create "hosts" table
CREATE TABLE `hosts` (
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
  `state` integer NULL,
  `cluster_id` integer NOT NULL,
  `host_policy_id` integer NULL,
  CONSTRAINT `fk_clusters_hosts` FOREIGN KEY (`cluster_id`) REFERENCES `clusters` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_host_policies_hosts` FOREIGN KEY (`host_policy_id`) REFERENCES `host_policies` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "hosts_name" to table: "hosts"
CREATE UNIQUE INDEX `hosts_name` ON `hosts` (`name`);
-- Create index "hosts_host_name" to table: "hosts"
CREATE UNIQUE INDEX `hosts_host_name` ON `hosts` (`host_name`);
-- Create index "hosts_mac" to table: "hosts"
CREATE UNIQUE INDEX `hosts_mac` ON `hosts` (`mac`);
-- Create index "idx_cluster_seq" to table: "hosts"
CREATE UNIQUE INDEX `idx_cluster_seq` ON `hosts` (`sequence_id`, `cluster_id`);
-- Create "maintenanceres_hosts" table
CREATE TABLE `maintenanceres_hosts` (
  `host_id` integer NULL,
  `maintenance_res_id` integer NULL,
  PRIMARY KEY (`host_id`, `maintenance_res_id`),
  CONSTRAINT `fk_maintenanceres_hosts_maintenance_res` FOREIGN KEY (`maintenance_res_id`) REFERENCES `maintenance_res` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_maintenanceres_hosts_host` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "users" table
CREATE TABLE `users` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `full_name` text NULL,
  `email` text NULL,
  `pass_hash` blob NULL
);
-- Create index "users_name" to table: "users"
CREATE UNIQUE INDEX `users_name` ON `users` (`name`);
-- Create index "users_email" to table: "users"
CREATE UNIQUE INDEX `users_email` ON `users` (`email`);
-- Create "groups" table
CREATE TABLE `groups` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `is_user_private` numeric NULL,
  `owner_id` integer NULL,
  CONSTRAINT `fk_groups_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "groups_name" to table: "groups"
CREATE UNIQUE INDEX `groups_name` ON `groups` (`name`);
-- Create "kickstarts" table
CREATE TABLE `kickstarts` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NULL,
  `filename` text NOT NULL,
  `owner_id` integer NULL,
  CONSTRAINT `fk_kickstarts_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "kickstarts_filename" to table: "kickstarts"
CREATE UNIQUE INDEX `kickstarts_filename` ON `kickstarts` (`filename`);
-- Create "distro_images" table
CREATE TABLE `distro_images` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `image_id` text NOT NULL,
  `type` text NOT NULL,
  `name` text NOT NULL,
  `kernel` text NULL,
  `initrd` text NULL,
  `iso` text NULL,
  `breed` text NULL,
  `local_boot` numeric NULL
);
-- Create index "distro_images_image_id" to table: "distro_images"
CREATE UNIQUE INDEX `distro_images_image_id` ON `distro_images` (`image_id`);
-- Create index "distro_images_name" to table: "distro_images"
CREATE UNIQUE INDEX `distro_images_name` ON `distro_images` (`name`);
-- Create "distros" table
CREATE TABLE `distros` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `is_default` numeric NULL,
  `description` text NULL,
  `owner_id` integer NULL,
  `distro_image_id` integer NULL,
  `kickstart_id` integer NULL,
  `kernel_args` text NULL,
  CONSTRAINT `fk_distro_images_distros` FOREIGN KEY (`distro_image_id`) REFERENCES `distro_images` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_distros_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_distros_kickstart` FOREIGN KEY (`kickstart_id`) REFERENCES `kickstarts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "distros_name" to table: "distros"
CREATE UNIQUE INDEX `distros_name` ON `distros` (`name`);
-- Create "profiles" table
CREATE TABLE `profiles` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `owner_id` integer NOT NULL,
  `distro_id` integer NULL,
  `is_default` numeric NULL,
  `kernel_args` text NULL,
  CONSTRAINT `fk_profiles_distro` FOREIGN KEY (`distro_id`) REFERENCES `distros` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_profiles_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_pname_owner" to table: "profiles"
CREATE UNIQUE INDEX `idx_pname_owner` ON `profiles` (`name`, `owner_id`);
-- Create "reservations" table
CREATE TABLE `reservations` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `owner_id` integer NULL,
  `group_id` integer NULL,
  `profile_id` integer NULL,
  `vlan` integer NULL,
  `start` datetime NULL,
  `end` datetime NULL,
  `orig_end` datetime NULL,
  `reset_end` datetime NULL,
  `extend_count` integer NULL,
  `installed` numeric NULL,
  `install_error` text NULL,
  `cycle_on_start` numeric NULL,
  `next_notify` integer NULL,
  `hash` text NOT NULL,
  CONSTRAINT `fk_reservations_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_groups_reservations` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_reservations_profile` FOREIGN KEY (`profile_id`) REFERENCES `profiles` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "reservations_name" to table: "reservations"
CREATE UNIQUE INDEX `reservations_name` ON `reservations` (`name`);
-- Create index "reservations_hash" to table: "reservations"
CREATE UNIQUE INDEX `reservations_hash` ON `reservations` (`hash`);
-- Create "reservations_hosts" table
CREATE TABLE `reservations_hosts` (
  `reservation_id` integer NULL,
  `host_id` integer NULL,
  PRIMARY KEY (`reservation_id`, `host_id`),
  CONSTRAINT `fk_reservations_hosts_host` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_reservations_hosts_reservation` FOREIGN KEY (`reservation_id`) REFERENCES `reservations` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "permissions" table
CREATE TABLE `permissions` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `group_id` integer NOT NULL,
  `fact` text NOT NULL,
  CONSTRAINT `fk_groups_permissions` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_group_fact" to table: "permissions"
CREATE UNIQUE INDEX `idx_group_fact` ON `permissions` (`group_id`, `fact`);
-- Create "bases" table
CREATE TABLE `bases` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL
);
-- Create "distros_groups" table
CREATE TABLE `distros_groups` (
  `group_id` integer NULL,
  `distro_id` integer NULL,
  PRIMARY KEY (`group_id`, `distro_id`),
  CONSTRAINT `fk_distros_groups_distro` FOREIGN KEY (`distro_id`) REFERENCES `distros` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_distros_groups_group` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "groups_users" table
CREATE TABLE `groups_users` (
  `group_id` integer NULL,
  `user_id` integer NULL,
  PRIMARY KEY (`group_id`, `user_id`),
  CONSTRAINT `fk_groups_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_groups_users_group` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "history_records" table
CREATE TABLE `history_records` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `hash` text NOT NULL,
  `status` text NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `owner` text NULL,
  `group` text NULL,
  `profile` text NULL,
  `distro` text NULL,
  `vlan` integer NULL,
  `start` datetime NULL,
  `end` datetime NULL,
  `orig_end` datetime NULL,
  `extend_count` integer NULL,
  `hosts` text NULL
);
-- Create "groups_policies" table
CREATE TABLE `groups_policies` (
  `host_policy_id` integer NULL,
  `group_id` integer NULL,
  PRIMARY KEY (`host_policy_id`, `group_id`),
  CONSTRAINT `fk_groups_policies_group` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_groups_policies_host_policy` FOREIGN KEY (`host_policy_id`) REFERENCES `host_policies` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
