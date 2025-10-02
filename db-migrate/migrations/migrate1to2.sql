-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Add column "kernel_info" to table: "distro_images"
ALTER TABLE `distro_images` ADD COLUMN `kernel_info` text NOT NULL DEFAULT '';
-- Add column "initrd_info" to table: "distro_images"
ALTER TABLE `distro_images` ADD COLUMN `initrd_info` text NOT NULL DEFAULT '';
PRAGMA foreign_keys = on;
