package com.fei.twitter.user;

import java.util.Optional;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;

public interface UserRepository extends JpaRepository<User, Long> {

    Optional<User> findByEmail(String email);

    /** Handles are unique regardless of case (idx_users_username_lower). */
    @Query("select u from User u where lower(u.username) = lower(?1)")
    Optional<User> findByUsernameIgnoreCase(String username);
}
