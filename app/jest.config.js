module.exports = {
  preset: 'jest-expo',
  testMatch: ['**/?(*.)+(spec|test).[tj]s?(x)'],
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  transformIgnorePatterns: [
    'node_modules/(?!(react-native|@react-native|expo(nent)?|expo-modules-core|@expo(nent)?/.*|expo-router|@expo/.*|@react-navigation/.*)/)',
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};
