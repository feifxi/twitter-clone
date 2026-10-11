package com.fei.twitter.user;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

/** The signed-in user's own profile: the public fields plus email and onboarding state. */
public record CurrentUserResponse(@JsonUnwrapped UserResponse profile, String email, boolean onboarded) {

    public static CurrentUserResponse of(User user) {
        return new CurrentUserResponse(UserResponse.of(UserView.of(user, false)), user.getEmail(), user.isOnboarded());
    }
}
