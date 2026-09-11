/** @type {import('jest').Config} */
const config = {
  // Test environment
  testEnvironment: 'jsdom',
  
  // Setup files
  setupFilesAfterEnv: ['<rootDir>/__tests__/setup.ts'],
  
  // Module name mapping
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
  },
  
  // Test file patterns
  testMatch: [
    '**/__tests__/**/*.(test|spec).(js|jsx|ts|tsx)',
    '**/*.(test|spec).(js|jsx|ts|tsx)'
  ],
  
  // Coverage configuration
  collectCoverageFrom: [
    'src/**/*.(js|jsx|ts|tsx)',
    '!src/**/*.d.ts',
    '!src/**/(*.stories|*.story).(js|jsx|ts|tsx)',
    '!src/app/**/layout.(js|jsx|ts|tsx)',
    '!src/app/**/page.(js|jsx|ts|tsx)',
  ],
  
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    }
  },
  
  // Transform files
  transform: {
    '^.+\\.(js|jsx|ts|tsx)$': ['babel-jest', { presets: ['next/babel'] }]
  },
  
  // Module file extensions
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx'],
  
  // Ignore patterns (legacy UI suites: need mocks for useCars / appointment hooks — re-enable when aligned with App Router)
  testPathIgnorePatterns: [
    '<rootDir>/.next/',
    '<rootDir>/node_modules/',
    '<rootDir>/__tests__/app/appointments/page.test.tsx',
    '<rootDir>/src/app/cars/__tests__/cars.test.tsx',
  ],
  
  // Clear mocks between tests
  clearMocks: true,
  
  // Verbose output
  verbose: true
};

module.exports = config;