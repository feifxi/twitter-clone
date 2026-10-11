package com.fei.twitter.user;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.ratelimit.RateLimit;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.shared.web.SuccessResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Users")
@Validated
@RestController
@RequestMapping("/api/v1/users")
class UserController {

    private final UserService users;

    UserController(UserService users) {
        this.users = users;
    }

    record UpdateProfileRequest(
            @Size(max = 50, message = "must be at most 50 characters") String displayName,
            @Size(max = 160, message = "must be at most 160 characters") String bio,
            @Size(max = 30, message = "must be at most 30 characters") String location,
            @Size(max = 100, message = "must be at most 100 characters")
            @Pattern(regexp = "^$|^https?://\\S+$", message = "must be a full http(s) URL") String website,
            @Size(max = 200, message = "must be at most 200 characters") String avatarKey,
            @Size(max = 200, message = "must be at most 200 characters") String bannerKey) {}

    record OnboardingRequest(
            @NotBlank(message = "this field is required")
            @Size(min = 3, max = 20, message = "must be 3-20 characters") String username,
            @NotBlank(message = "this field is required")
            @Size(max = 50, message = "must be at most 50 characters") String displayName) {}

    @Operation(summary = "A user's profile")
    @GetMapping("/{id}")
    UserResponse get(@PathVariable long id, Viewer viewer) {
        return UserResponse.of(users.get(id, viewer.id()));
    }

    @Operation(summary = "A user's profile by handle (case-insensitive)")
    @GetMapping("/by-username/{username}")
    UserResponse getByUsername(@PathVariable @Size(max = 20, message = "must be at most 20 characters") String username, Viewer viewer) {
        return UserResponse.of(users.getByUsername(username, viewer.id()));
    }

    @Operation(summary = "Accounts following the user, most recent first")
    @GetMapping("/{id}/followers")
    PageResponse<UserResponse> followers(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(users.followers(id, page, viewer.id()), UserResponse::of);
    }

    @Operation(summary = "Accounts the user follows, most recent first")
    @GetMapping("/{id}/following")
    PageResponse<UserResponse> following(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(users.following(id, page, viewer.id()), UserResponse::of);
    }

    @Operation(summary = "Edit your profile (omitted fields are unchanged, empty strings clear)")
    @RateLimit(RateLimit.Policy.WRITE)
    @PutMapping("/profile")
    UserResponse updateProfile(@Valid @RequestBody UpdateProfileRequest r, Viewer viewer) {
        User user = users.updateProfile(viewer.requireId(),
                new UserService.ProfileUpdate(r.displayName(), r.bio(), r.location(), r.website(), r.avatarKey(), r.bannerKey()));
        return UserResponse.of(UserView.of(user, false));
    }

    @Operation(summary = "Choose your username (first sign-in)")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/me/onboarding")
    CurrentUserResponse completeOnboarding(@Valid @RequestBody OnboardingRequest r, Viewer viewer) {
        return CurrentUserResponse.of(users.completeOnboarding(viewer.requireId(), r.username().strip(), r.displayName()));
    }

    @Operation(summary = "Follow a user")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/{id}/follow")
    SuccessResponse follow(@PathVariable long id, Viewer viewer) {
        users.follow(viewer.requireId(), id);
        return SuccessResponse.OK;
    }

    @Operation(summary = "Unfollow a user")
    @RateLimit(RateLimit.Policy.WRITE)
    @DeleteMapping("/{id}/follow")
    SuccessResponse unfollow(@PathVariable long id, Viewer viewer) {
        users.unfollow(viewer.requireId(), id);
        return SuccessResponse.OK;
    }
}
