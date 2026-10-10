import type { CurrentUserResponse } from './user';

export interface GoogleAuthRequest {
    idToken: string;
}

export interface AuthResponse {
    accessToken: string;
    refreshToken: string;
    user: CurrentUserResponse;
}
