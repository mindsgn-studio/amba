import { useMemo, useState } from 'react';

import {
  RecordingPresets,
  requestRecordingPermissionsAsync,
  setAudioModeAsync,
  useAudioRecorder,
  useAudioRecorderState,
} from 'expo-audio';

import { saveReadingSession } from '@/db/client';
import { scoreReadingSession } from '@/features/reading/scoring';
import { track } from '@/services/analytics';
import { transcribeRecording } from '@/services/transcription';
import type { ReadingPassage, ScorePayload, SessionControllerState, UserProfile } from '@/types/models';

const createSessionId = () =>
  `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

export function useReadingSession(
  passage: ReadingPassage,
  profile: UserProfile | null,
) {
  const recorder = useAudioRecorder(RecordingPresets.HIGH_QUALITY);
  const recorderState = useAudioRecorderState(recorder, 250);
  const [state, setState] = useState<SessionControllerState>('idle');
  const [error, setError] = useState<string | null>(null);
  const [score, setScore] = useState<ScorePayload | null>(null);

  const progressWords = useMemo(() => {
    if (!recorderState.durationMillis) {
      return 0;
    }

    const durationSec = recorderState.durationMillis / 1000;
    const fraction = Math.min(durationSec / passage.estimatedDurationSec, 1);
    return Math.max(1, Math.round(fraction * passage.wordCount));
  }, [passage.estimatedDurationSec, passage.wordCount, recorderState.durationMillis]);

  const start = async () => {
    setError(null);
    setState('preparing');

    try {
      const permission = await requestRecordingPermissionsAsync();

      if (!permission.granted) {
        setState('error');
        setError('Microphone permission is required before you can start reading.');
        return null;
      }

      await setAudioModeAsync({
        allowsRecording: true,
        playsInSilentMode: true,
      });

      await recorder.prepareToRecordAsync();
      recorder.record();
      setState('recording');
      track('reading_started', {
        passageId: passage.id,
        difficulty: passage.difficulty,
      });
      return true;
    } catch (startError) {
      setState('error');
      setError(
        startError instanceof Error
          ? startError.message
          : 'We could not start the recording session.',
      );
      return null;
    }
  };

  const stop = async () => {
    setState('processing');

    try {
      await recorder.stop();

      if (!recorder.uri) {
        throw new Error('No recording file was created.');
      }

      const transcript = await transcribeRecording({
        fileUri: recorder.uri,
        expectedText: passage.text,
      });

      const payload = scoreReadingSession({
        passage,
        transcript,
        durationSec: Math.max(1, Math.round(recorderState.durationMillis / 1000)),
        sessionId: createSessionId(),
        userId: profile?.uid ?? null,
      });

      await saveReadingSession(payload);
      track('reading_completed', {
        passageId: passage.id,
        accuracy: payload.accuracy,
        wordsCorrect: payload.wordsCorrect,
      });
      track('session_scored', {
        sessionId: payload.id,
        xpEarned: payload.xpEarned,
      });

      setScore(payload);
      setState('scored');
      return payload;
    } catch (stopError) {
      setState('error');
      setError(
        stopError instanceof Error
          ? stopError.message
          : 'We could not score this reading session.',
      );
      return null;
    }
  };

  return {
    state,
    error,
    score,
    progressWords,
    recorderState,
    isRecording: recorderState.isRecording,
    start,
    stop,
  };
}
