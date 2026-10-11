import axios from 'axios';
import { axiosInstance } from './axiosInstance';

/**
 * Uploads a file straight to S3: the API presigns a PUT for exactly this type and
 * size, the browser sends the bytes. Returns the object key to send to the API.
 */
export async function uploadFileWithPresignedUrl(
  file: File,
  folder: 'tweets' | 'avatars' | 'banners'
): Promise<string> {
  const { data } = await axiosInstance.post<{ presignedUrl: string; objectKey: string }>(
    '/uploads/presign',
    {
      contentType: file.type,
      folder,
      contentLength: file.size,
    }
  );

  // Plain axios: S3 must not get the API's credentials or its interceptors.
  await axios.put(data.presignedUrl, file, {
    headers: {
      'Content-Type': file.type,
    },
  });

  return data.objectKey;
}
