import type { ExpoConfig } from "expo/config";

const config: ExpoConfig = {
  name: "Horologia",
  slug: "horologia",
  version: "0.0.0",
  scheme: "horologia",
  userInterfaceStyle: "automatic",
  icon: "./assets/images/icon.png",
  ios: {
    bundleIdentifier: "dev.horologia.mobile",
    supportsTablet: true,
    requireFullScreen: false,
    icon: "./assets/expo.icon",
  },
  android: {
    package: "dev.horologia.mobile",
    adaptiveIcon: {
      backgroundColor: "#E7F0EA",
      foregroundImage: "./assets/images/android-icon-foreground.png",
      backgroundImage: "./assets/images/android-icon-background.png",
      monochromeImage: "./assets/images/android-icon-monochrome.png",
    },
    predictiveBackGestureEnabled: true,
  },
  plugins: [
    "expo-router",
    "expo-secure-store",
    "expo-sqlite",
    "expo-status-bar",
    "expo-web-browser",
    [
      "expo-splash-screen",
      {
        backgroundColor: "#F6FAF7",
        image: "./assets/images/splash-icon.png",
        imageWidth: 76,
      },
    ],
  ],
  experiments: {
    reactCompiler: true,
    typedRoutes: true,
  },
};

export default config;
