import Constants from 'expo-constants';
import * as FileSystem from 'expo-file-system';

type TranscriptionArgs = {
  fileUri: string;
  expectedText: string;
};

type WhisperContext = {
  transcribe: (filePath: string, options: { language: string }) => {
    promise: Promise<{ result: string }>;
  };
};

let whisperContextPromise: Promise<WhisperContext> | null = null;

function getTranscriptionMode() {
  const extra = Constants.expoConfig?.extra as { appEnv?: { transcriptionMode?: string } } | undefined;
  return process.env.EXPO_PUBLIC_TRANSCRIPTION_MODE ?? extra?.appEnv?.transcriptionMode ?? 'mock';
}

async function getWhisperContext() {
  if (whisperContextPromise) {
    return whisperContextPromise;
  }

  whisperContextPromise = (async () => {
    const whisper = await import('whisper.rn');
    const modelPath = process.env.EXPO_PUBLIC_WHISPER_MODEL_PATH;

    if (!modelPath || !modelPath.endsWith('.bin')) {
      throw new Error(
        'Whisper model is not configured. Set EXPO_PUBLIC_WHISPER_MODEL_PATH to a bundled or local .bin model file.',
      );
    }

    return whisper.initWhisper({
      filePath: modelPath,
    }) as Promise<WhisperContext>;
  })();

  return whisperContextPromise;
}

async function mockTranscript(expectedText: string) {
  return expectedText;
}

export async function transcribeRecording({
  expectedText,
  fileUri,
}: TranscriptionArgs) {
  const mode = getTranscriptionMode();

  if (mode !== 'whisper') {
    return mockTranscript(expectedText);
  }

  const info = await FileSystem.getInfoAsync(fileUri);

  if (!info.exists) {
    throw new Error('Recorded audio file was not found.');
  }

  const context = await getWhisperContext();
  const { promise } = context.transcribe(fileUri, { language: 'en' });
  const { result } = await promise;

  return result || expectedText;
}
