
export interface paths {
    "/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["LoginRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/register": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["RegisterRequest"];
                };
            };
            responses: {
                201: components["responses"]["OK"];
                202: components["responses"]["OK"];
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/me": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/role-templates": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/permission-requests": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["PermissionRequestSubmitRequest"];
                };
            };
            responses: {
                202: components["responses"]["OK"];
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/public/ihks": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    query?: string;
                    state?: string;
                    includePending?: boolean;
                    limit?: number;
                    cursor?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["PublicIHKListResponse"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/public/ihks/{slug}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    slug: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["PublicIHK"];
                    };
                };
                404: components["responses"]["NotFound"];
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/public/info-suggestions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            ok: boolean;
                            user: Record<string, never>;
                            effectiveMask: number;
                            abilities: {
                                canHidePendingHints: boolean;
                                canManageModerationTerms: boolean;
                                canManagePermissionRequests?: boolean;
                            };
                        };
                    };
                };
            };
        };
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["SubmitInfoSuggestionRequest"];
                };
            };
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["OKMessage"];
                    };
                };
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["SubmitError"];
                    };
                };
                429: components["responses"]["RateLimited"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    status?: components["schemas"]["SuggestionStatus"];
                    ihkId?: string;
                    publicPendingVisible?: boolean;
                    limit?: number;
                    cursor?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/start-review": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/accept": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/reject": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/needs-more-info": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/mark-spam": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/reopen": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/hide-pending": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        reason: string;
                    };
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/info-suggestions/{id}/apply": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["ApplyInfoSuggestionRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/moderation-terms": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                201: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/moderation-terms/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        options?: never;
        head?: never;
        patch: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        trace?: never;
    };
    "/admin/audit-logs": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    limit?: number;
                    cursor?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/users": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    query?: string;
                    limit?: number;
                    cursor?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["CreateAdminUserRequest"];
                };
            };
            responses: {
                201: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/role-templates": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/users/{id}/roles": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["AssignUserRoleRequest"];
                };
            };
            responses: {
                201: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/users/{id}/roles/{assignmentId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                    assignmentId: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: components["responses"]["OK"];
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/permission-requests": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    status?: "pending" | "approved" | "rejected";
                    limit?: number;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/permission-requests/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/permission-requests/{id}/approve": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: {
                content: {
                    "application/json": components["schemas"]["PermissionRequestDecisionRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/permission-requests/{id}/reject": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: {
                content: {
                    "application/json": components["schemas"]["PermissionRequestDecisionRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/ihks": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: {
                    query?: string;
                    state?: string;
                    limit?: number;
                    cursor?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/ihks/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["UpdateAdminIHKRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        trace?: never;
    };
    "/admin/ihks/{id}/info/versions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/ihks/{id}/info/publish": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["PublishIHKInfoRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/ihks/{id}/info/rollback": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    id: components["parameters"]["ID"];
                };
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": components["schemas"]["RollbackIHKInfoRequest"];
                };
            };
            responses: {
                200: components["responses"]["OK"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        OK: {
            ok: boolean;
        };
        OKMessage: {
            ok: boolean;
            message: string;
        };
        LoginRequest: {
            email: string;
            password: string;
        };
        RegisterRequest: {
            email: string;
            displayName: string;
            password: string;
            requestedRoleTemplateId?: string;
            requestedScopeType?: "global" | "state" | "ihk";
            requestedScopeId?: string;
            proofFileName?: string;
            proofMimeType?: string;
            proofContentBase64?: string;
            proofNote?: string;
        };
        PermissionRequestSubmitRequest: {
            requestedRoleTemplateId: string;
            requestedScopeType: "global" | "state" | "ihk";
            requestedScopeId?: string;
            proofFileName?: string;
            proofMimeType?: string;
            proofContentBase64?: string;
            proofNote?: string;
        };
        PermissionRequestDecisionRequest: {
            note?: string;
        };
        SubmitError: {
            ok: false;
            code?: "LANGUAGE_NOT_GERMAN" | "WORD_FILTER_BLOCKED" | "HTML_BLOCKED" | "URL_BLOCKED" | "LENGTH_BLOCKED";
            message: string;
        };
        PublicIHKListResponse: {
            items: components["schemas"]["PublicIHK"][];
            nextCursor: string | null;
        };
        PublicIHK: {
            id: string;
            name: string;
            slug: string;
            city?: string | null;
            state: string;
            officialUrl?: string | null;
            info: {
                currentText: string;
                confidenceLevel: "low" | "medium" | "high";
                sourceSummary?: string | null;
                updatedAt: string;
            };
            pendingHints: components["schemas"]["PendingHint"][];
        };
        PendingHint: {
            id: string;
            publicPendingText: string;
            sourceUrl?: string | null;
            sourceNote?: string | null;
            createdAt: string;
        };
        SubmitInfoSuggestionRequest: {
            ihkId: string;
            suggestedChange: string;
            reason?: string;
            sourceUrl?: string;
            sourceNote?: string;
            submittedEmail?: string;
            honeypot: string;
        };
        ApplyInfoSuggestionRequest: {
            newText: string;
            confidenceLevel: "low" | "medium" | "high";
            sourceSummary?: string;
            changeSummary: string;
        };
        UpdateAdminIHKRequest: {
            officialUrl?: string | null;
        };
        PublishIHKInfoRequest: {
            newText: string;
            confidenceLevel: "low" | "medium" | "high";
            sourceSummary?: string;
            changeSummary: string;
        };
        RollbackIHKInfoRequest: {
            versionId: string;
            confidenceLevel?: "low" | "medium" | "high";
            sourceSummary?: string;
            changeSummary: string;
        };
        AssignUserRoleRequest: {
            roleTemplateId: string;
            scopeType: "global" | "state" | "ihk";
            scopeId?: string;
            expiresAt?: string;
        };
        CreateAdminUserRequest: {
            email: string;
            displayName: string;
            password: string;
            roleTemplateId?: string;
            scopeType?: "global" | "state" | "ihk";
            scopeId?: string;
        };
        SuggestionStatus: "submitted" | "under_review" | "needs_more_info" | "accepted" | "rejected" | "applied" | "archived" | "spam";
    };
    responses: {
        OK: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["OK"];
            };
        };
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content?: never;
        };
        RateLimited: {
            headers: {
                [name: string]: unknown;
            };
            content?: never;
        };
    };
    parameters: {
        ID: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export type operations = Record<string, never>;
