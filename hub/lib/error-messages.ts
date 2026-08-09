import { ApiRequestError } from "@/lib/api-client";

export interface ErrorCopy {
  title: string;
  message: string;
  hint?: string;
}

export function getErrorMessage(err: unknown): ErrorCopy {
  if (err instanceof ApiRequestError) {
    switch (err.status) {
      case 0:
        return {
          title: "Can't reach the server",
          message: "The server may be down, or your network connection is unavailable.",
          hint: "Check that RenOS Core is running, then try again.",
        };
      case 401:
        return {
          title: "Your session has expired",
          message: "Sign in again to continue.",
          hint: "Your session ended. Sign in again to keep working.",
        };
      case 403:
        return {
          title: "You don't have permission",
          message: "Your role doesn't allow this action.",
          hint: "Contact an administrator or request a higher role.",
        };
      case 404:
        return {
          title: "Not found",
          message: "This item may have been deleted.",
        };
      case 429:
        return {
          title: "Too many requests",
          message: "Slow down — you're making requests too quickly.",
          hint: "Wait a moment and try again.",
        };
      case 400:
      case 422:
        return {
          title: "Invalid input",
          message: err.message || "Some of the provided values are invalid.",
          hint: "Check your input and try again.",
        };
      case 500:
      case 502:
      case 503:
        return {
          title: "Server error",
          message: "RenOS hit an unexpected problem.",
          hint: "Check /var/log/renos/install.log or contact support. If it keeps happening, try again in a moment.",
        };
      default:
        return {
          title: "Something went wrong",
          message: err.message || "An unexpected error occurred.",
        };
    }
  }

  if (err instanceof Error) {
    return {
      title: "Something went wrong",
      message: err.message || "An unexpected error occurred.",
    };
  }

  return {
    title: "Something went wrong",
    message: "An unexpected error occurred.",
  };
}
