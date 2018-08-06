module.exports = {
    "collectCoverageFrom": [
      "src/**/*.{js,jsx,mjs}"
    ],
    "setupFiles": [
      "<rootDir>/config/polyfills.js"
    ],
    "setupTestFrameworkScriptFile": "<rootDir>/config/setupTests.js",
    "testMatch": [
      "<rootDir>/src/**/__tests__/**/*.{js,jsx,mjs}",
      "<rootDir>/src/**/?(*.)(spec|test).{js,jsx,mjs}"
    ],
    "testEnvironment": "node",
    "testURL": "http://localhost",
    "transform": {
      "^.+\\.(js|jsx|mjs)$": "<rootDir>/node_modules/babel-jest",
      "^.+\\.css$": "<rootDir>/config/jest/cssTransform.js",
      "^(?!.*\\.(js|jsx|mjs|css|json)$)": "<rootDir>/config/jest/fileTransform.js"
    },
    "transformIgnorePatterns": [
      "[/\\\\]node_modules[/\\\\].+\\.(js|jsx|mjs)$",
    ],
    "snapshotSerializers": ["enzyme-to-json/serializer"],
    "moduleFileExtensions": [
      "web.js",
      "js",
      "json",
      "web.jsx",
      "jsx",
      "node",
      "mjs"
    ]
  }
