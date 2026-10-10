import axios from 'axios';
import { axiosInstance } from './axiosInstance';

/**
 * Handles the 2-step presigned S3 upload process:
 * 1. Request presigned URL from our API
 * 2. Upload the file bytes directly to S3 via PUT
 * Returns the objectKey to be sent in the final API request.
 */
export async function uploadFileWithPresignedUrl(
  file: File,
  folder: 'tweets' | 'avatars' | 'banners'
): Promise<string> {
  // 1. Get presigned URL
  const { data } = await axiosInstance.post<{ presignedUrl: string; objectKey: string }>(
    '/uploads/presign',
    {
      contentType: file.type,
      folder,
      contentLength: file.size,
    }
  );

  // 2. PUT file directly to S3 URL
  // We use bare axios here so our interceptors (like auth headers) don't get appended to the S3 request
  await axios.put(data.presignedUrl, file, {
    headers: {
      'Content-Type': file.type,
    },
  });

  return data.objectKey;
}
