import { z } from 'zod';

export const tweetContentSchema = z
  .string()
  .max(280, 'Tweet content must be under 280 characters')
  .optional();

export const tweetRequestSchema = z.object({
  content: z.string().max(280, 'Tweet content must be under 280 characters').optional(),
  parentId: z.number().nullable().optional(),
});

/** Same limits as the API's profile validation. */
export const updateProfileSchema = z.object({
  displayName: z.string().trim().min(1, "Name can't be blank").max(50, 'Name can be at most 50 characters'),
  bio: z.string().max(160, 'Bio can be at most 160 characters'),
  location: z.string().max(30, 'Location can be at most 30 characters'),
  website: z
    .string()
    .max(100, 'Website can be at most 100 characters')
    .refine((v) => v === '' || /^https?:\/\/[^\s.]+\.[^\s]+$/.test(v), 'Enter a full URL, like https://example.com'),
});

export const googleAuthRequestSchema = z.object({
  idToken: z.string().min(1, 'Token is required'),
});

export type TweetRequestInput = z.infer<typeof tweetRequestSchema>;
export type UpdateProfileInput = z.infer<typeof updateProfileSchema>;
export type GoogleAuthRequestInput = z.infer<typeof googleAuthRequestSchema>;
