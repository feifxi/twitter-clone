package com.fei.twitter.notification;

/** Published inside the creating transaction; pushed to the recipient after it commits. */
public record NotificationCreated(long notificationId, long recipientId) {}
