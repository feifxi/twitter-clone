package com.fei.twitter.shared.error;

import com.fei.twitter.shared.web.RequestIdFilter;
import jakarta.servlet.http.HttpServletRequest;
import java.sql.SQLException;
import java.util.List;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.web.HttpRequestMethodNotSupportedException;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.MissingServletRequestParameterException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.HandlerMethodValidationException;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.servlet.resource.NoResourceFoundException;

/**
 * Maps every failure to the API's error body. Only {@link ApiException} messages reach
 * clients; anything unexpected becomes an opaque 500 and is logged.
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger log = LoggerFactory.getLogger(GlobalExceptionHandler.class);

    @ExceptionHandler(ValidationException.class)
    ResponseEntity<ErrorResponse> validation(ValidationException e, HttpServletRequest request) {
        return validationError(request, List.of(new ErrorResponse.FieldError(e.field(), e.getMessage())));
    }

    @ExceptionHandler(ApiException.class)
    ResponseEntity<ErrorResponse> api(ApiException e, HttpServletRequest request) {
        return respond(e.code(), e.getMessage(), request);
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    ResponseEntity<ErrorResponse> invalidBody(MethodArgumentNotValidException e, HttpServletRequest request) {
        List<ErrorResponse.FieldError> details = e.getBindingResult().getFieldErrors().stream()
                .map(fe -> new ErrorResponse.FieldError(fe.getField(), fe.getDefaultMessage()))
                .toList();
        return validationError(request, details);
    }

    @ExceptionHandler(HandlerMethodValidationException.class)
    ResponseEntity<ErrorResponse> invalidParameter(HandlerMethodValidationException e, HttpServletRequest request) {
        List<ErrorResponse.FieldError> details = e.getParameterValidationResults().stream()
                .flatMap(r -> r.getResolvableErrors().stream()
                        .map(err -> new ErrorResponse.FieldError(r.getMethodParameter().getParameterName(), err.getDefaultMessage())))
                .toList();
        return validationError(request, details);
    }

    @ExceptionHandler(MethodArgumentTypeMismatchException.class)
    ResponseEntity<ErrorResponse> typeMismatch(MethodArgumentTypeMismatchException e, HttpServletRequest request) {
        return validationError(request, List.of(new ErrorResponse.FieldError(e.getName(), "must be a valid number")));
    }

    @ExceptionHandler(MissingServletRequestParameterException.class)
    ResponseEntity<ErrorResponse> missingParameter(MissingServletRequestParameterException e, HttpServletRequest request) {
        return validationError(request, List.of(new ErrorResponse.FieldError(e.getParameterName(), "this field is required")));
    }

    @ExceptionHandler(HttpMessageNotReadableException.class)
    ResponseEntity<ErrorResponse> unreadable(HttpMessageNotReadableException e, HttpServletRequest request) {
        return respond(ErrorCode.BAD_REQUEST, "malformed JSON request body", request);
    }

    @ExceptionHandler({NoResourceFoundException.class, HttpRequestMethodNotSupportedException.class})
    ResponseEntity<ErrorResponse> noRoute(Exception e, HttpServletRequest request) {
        return respond(ErrorCode.NOT_FOUND, "route not found", request);
    }

    @ExceptionHandler(AccessDeniedException.class)
    ResponseEntity<ErrorResponse> denied(AccessDeniedException e, HttpServletRequest request) {
        return respond(ErrorCode.FORBIDDEN, "forbidden", request);
    }

    /** Constraint violations that reach a controller are client errors, not 500s. */
    @ExceptionHandler(DataIntegrityViolationException.class)
    ResponseEntity<ErrorResponse> integrity(DataIntegrityViolationException e, HttpServletRequest request) {
        return switch (sqlState(e)) {
            case "23505" -> respond(ErrorCode.CONFLICT, "resource already exists", request);
            case "23503" -> respond(ErrorCode.BAD_REQUEST, "referenced resource does not exist", request);
            case "23514" -> respond(ErrorCode.BAD_REQUEST, "invalid request", request);
            default -> unexpected(e, request);
        };
    }

    @ExceptionHandler(Exception.class)
    ResponseEntity<ErrorResponse> unexpected(Exception e, HttpServletRequest request) {
        log.error("request failed: {} {}", request.getMethod(), request.getRequestURI(), e);
        return respond(ErrorCode.INTERNAL_ERROR, "internal server error", request);
    }

    private static ResponseEntity<ErrorResponse> validationError(HttpServletRequest request, List<ErrorResponse.FieldError> details) {
        return ResponseEntity.status(ErrorCode.VALIDATION_ERROR.status())
                .body(new ErrorResponse(ErrorCode.VALIDATION_ERROR.name(), "invalid request payload", RequestIdFilter.current(request), details));
    }

    static ResponseEntity<ErrorResponse> respond(ErrorCode code, String message, HttpServletRequest request) {
        return ResponseEntity.status(code.status()).body(ErrorResponse.of(code, message, RequestIdFilter.current(request)));
    }

    static String sqlState(Throwable e) {
        for (Throwable t = e; t != null; t = t.getCause()) {
            if (t instanceof SQLException sql && sql.getSQLState() != null) {
                return sql.getSQLState();
            }
        }
        return "";
    }
}
