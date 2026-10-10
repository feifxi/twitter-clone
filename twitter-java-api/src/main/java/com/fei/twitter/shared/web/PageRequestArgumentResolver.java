package com.fei.twitter.shared.web;

import com.fei.twitter.shared.error.ValidationException;
import org.springframework.core.MethodParameter;
import org.springframework.stereotype.Component;
import org.springframework.web.bind.support.WebDataBinderFactory;
import org.springframework.web.context.request.NativeWebRequest;
import org.springframework.web.method.support.HandlerMethodArgumentResolver;
import org.springframework.web.method.support.ModelAndViewContainer;

/** Resolves a {@link PageRequest} handler parameter from {@code ?cursor=&size=} (default 20, capped at 50). */
@Component
public class PageRequestArgumentResolver implements HandlerMethodArgumentResolver {

    @Override
    public boolean supportsParameter(MethodParameter parameter) {
        return parameter.getParameterType() == PageRequest.class;
    }

    @Override
    public PageRequest resolveArgument(
            MethodParameter parameter, ModelAndViewContainer mav, NativeWebRequest request, WebDataBinderFactory binders) {
        int limit = PageRequest.DEFAULT_LIMIT;
        String size = request.getParameter("size");
        if (size != null && !size.isBlank()) {
            try {
                limit = Integer.parseInt(size.trim());
            } catch (NumberFormatException e) {
                throw new ValidationException("size", "must be a valid number");
            }
            if (limit < 1) {
                throw new ValidationException("size", "must be at least 1");
            }
            limit = Math.min(limit, PageRequest.MAX_LIMIT);
        }

        Cursor after = null;
        String cursor = request.getParameter("cursor");
        if (cursor != null && !cursor.isBlank()) {
            try {
                after = Cursor.decode(cursor.trim());
            } catch (IllegalArgumentException e) {
                throw new ValidationException("cursor", "invalid cursor");
            }
        }
        return new PageRequest(after, limit);
    }
}
