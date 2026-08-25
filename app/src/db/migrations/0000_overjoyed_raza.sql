CREATE TABLE `books` (
	`id` text PRIMARY KEY NOT NULL,
	`title` text NOT NULL,
	`author` text,
	`language` text DEFAULT 'en' NOT NULL,
	`source_type` text DEFAULT 'text' NOT NULL,
	`status` text DEFAULT 'queued' NOT NULL,
	`created_at` text NOT NULL,
	`updated_at` text NOT NULL
);
--> statement-breakpoint
CREATE TABLE `chapters` (
	`id` text PRIMARY KEY NOT NULL,
	`book_id` text NOT NULL,
	`chapter_index` integer NOT NULL,
	`title` text NOT NULL,
	FOREIGN KEY (`book_id`) REFERENCES `books`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE INDEX `chapters_book_id_idx` ON `chapters` (`book_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `chapters_book_index_unique` ON `chapters` (`book_id`,`chapter_index`);--> statement-breakpoint
CREATE TABLE `chunks` (
	`id` text PRIMARY KEY NOT NULL,
	`chapter_id` text NOT NULL,
	`chunk_index` integer NOT NULL,
	`duration` real,
	`audio_url` text,
	`timing_url` text,
	FOREIGN KEY (`chapter_id`) REFERENCES `chapters`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE INDEX `chunks_chapter_id_idx` ON `chunks` (`chapter_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `chunks_chapter_index_unique` ON `chunks` (`chapter_id`,`chunk_index`);--> statement-breakpoint
CREATE TABLE `jobs` (
	`id` text PRIMARY KEY NOT NULL,
	`book_id` text NOT NULL,
	`status` text DEFAULT 'queued' NOT NULL,
	`progress` real DEFAULT 0 NOT NULL,
	`error` text,
	`created_at` text NOT NULL,
	`updated_at` text NOT NULL,
	FOREIGN KEY (`book_id`) REFERENCES `books`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `reading_progress` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`book_id` text NOT NULL,
	`chapter_id` text NOT NULL,
	`chunk_id` text NOT NULL,
	`position_seconds` real DEFAULT 0 NOT NULL,
	`word_index` integer DEFAULT 0 NOT NULL,
	`completed` integer DEFAULT 0 NOT NULL,
	`updated_at` text NOT NULL,
	FOREIGN KEY (`book_id`) REFERENCES `books`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `reading_progress_book_unique` ON `reading_progress` (`book_id`);