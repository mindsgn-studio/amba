import Constants from 'expo-constants';
import { initializeApp, getApps, getApp } from 'firebase/app';

type FirebaseConfig = {
  apiKey?: string;
  authDomain?: string;
  projectId?: string;
  storageBucket?: string;
  messagingSenderId?: string;
  appId?: string;
  measurementId?: string;
};

function getFirebaseConfig(): FirebaseConfig {
  const extra = Constants.expoConfig?.extra as { firebase?: FirebaseConfig } | undefined;

  return {
    apiKey: process.env.EXPO_PUBLIC_FIREBASE_API_KEY ?? extra?.firebase?.apiKey,
    authDomain: process.env.EXPO_PUBLIC_FIREBASE_AUTH_DOMAIN ?? extra?.firebase?.authDomain,
    projectId: process.env.EXPO_PUBLIC_FIREBASE_PROJECT_ID ?? extra?.firebase?.projectId,
    storageBucket: process.env.EXPO_PUBLIC_FIREBASE_STORAGE_BUCKET ?? extra?.firebase?.storageBucket,
    messagingSenderId:
      process.env.EXPO_PUBLIC_FIREBASE_MESSAGING_SENDER_ID ?? extra?.firebase?.messagingSenderId,
    appId: process.env.EXPO_PUBLIC_FIREBASE_APP_ID ?? extra?.firebase?.appId,
    measurementId:
      process.env.EXPO_PUBLIC_FIREBASE_MEASUREMENT_ID ?? extra?.firebase?.measurementId,
  };
}

function canInitializeFirebase(config: FirebaseConfig) {
  return Boolean(config.apiKey && config.appId && config.projectId);
}

const firebaseConfig = getFirebaseConfig();

export const firebaseApp = canInitializeFirebase(firebaseConfig)
  ? getApps().length
    ? getApp()
    : initializeApp(firebaseConfig)
  : null;

export function isFirebaseConfigured() {
  return firebaseApp !== null;
}
