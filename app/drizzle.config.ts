import type { Config } from 'drizzle-kit';
import { defineConfig } from "drizzle-kit";

export default defineConfig ({
  schema: './src/db/schema.ts',
  out: './src/db/migrations',
  dialect: 'sqlite',
  driver: 'expo',
}) satisfies Config;
