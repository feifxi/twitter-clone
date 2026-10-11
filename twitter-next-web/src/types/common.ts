export interface ValidationError {
    field: string;
    message: string;
}

/** Backend error payload. Use fieldErrors for form binding. */
export interface ErrorResponse {
    code: string;
    message: string;
    details?: ValidationError[];
}

/** Map of field name -> message for easy form input binding. */
export type FieldErrors = Record<string, string>;

export interface PageResponse<T> {
    items: T[];
    hasNext: boolean;
    /** Opaque; pass back as ?cursor=. null on the last page. */
    nextCursor: string | null;
}

export interface ApiResponse {
    success: boolean;
    message: string;
}
