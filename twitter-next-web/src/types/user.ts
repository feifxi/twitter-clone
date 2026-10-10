export type Role = 'USER' | 'ADMIN';

export interface UserResponse {
    id: number;
    username: string;
    displayName: string | null;
    avatarUrl: string | null;
    bannerUrl: string | null;
    bio: string | null;
    followersCount: number;
    followingCount: number;
    isFollowing: boolean;
}

/** The signed-in user's own profile; only this shape includes private fields. */
export interface CurrentUserResponse extends UserResponse {
    email: string;
}

export interface UpdateProfileRequest {
    displayName?: string;
    bio?: string;
}
