import { getAuth, signInAnonymously } from 'firebase/auth';

import { firebaseApp } from '@/services/firebase';
import type { UserProfile } from '@/types/models';

let authInstance: ReturnType<typeof getAuth> | null = null;

function getFirebaseAuth() {
  if (!firebaseApp) {
    return null;
  }

  if (authInstance) {
    return authInstance;
  }

  authInstance = getAuth(firebaseApp);

  return authInstance;
}

export async function ensureAnonymousSession(): Promise<UserProfile> {
  const auth = getFirebaseAuth();

  if (!auth) {
    return {
      uid: 'local-guest',
      isAnonymous: true,
      createdAt: Date.now(),
      mode: 'local',
    };
  }

  if (auth.currentUser) {
    return {
      uid: auth.currentUser.uid,
      isAnonymous: auth.currentUser.isAnonymous,
      createdAt: auth.currentUser.metadata.creationTime
        ? new Date(auth.currentUser.metadata.creationTime).getTime()
        : Date.now(),
      mode: 'firebase',
    };
  }

  const credential = await signInAnonymously(auth);

  return {
    uid: credential.user.uid,
    isAnonymous: credential.user.isAnonymous,
    createdAt: credential.user.metadata.creationTime
      ? new Date(credential.user.metadata.creationTime).getTime()
      : Date.now(),
    mode: 'firebase',
  };
}
