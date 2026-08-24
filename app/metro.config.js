// Default Expo Metro configuration. Drizzle migrations are inlined into
// TypeScript by tools/build-migrations.mjs, so no custom asset handling
// is required here.
const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

module.exports = config;
