export type Role = 'USER' | 'ADMIN';

export interface UserResponse {
    id: number;
    username: string;
    displayName: string | null;
    avatarUrl: string | null;
    bannerUrl: string | null;
    bio: string | null;
    location: string | null;
    website: string | null;
    followersCount: number;
    followingCount: number;
    isFollowing: boolean;
    createdAt: string;
}

/** The signed-in user's own profile; only this shape includes private fields. */
export interface CurrentUserResponse extends UserResponse {
    email: string;
    /** False until the user has picked their own username. */
    onboarded: boolean;
}

export interface UpdateProfileRequest {
    displayName?: string;
    bio?: string;
}
