# extras-v1 — 01: Global Settings — Profile & Password

## What to build

A user-level **Global Settings** page where the user manages their own profile. Three sections:

**Avatar:** A set of client-side PNG avatars (stored as static assets in the frontend). The user picks one and the filename is sent to the backend and stored on the user record. No file uploads, no external avatar service.

**Profile info:** Display name and email fields. Editable inline. Backend persists changes.

**Password change:** Old password → new password → confirm new password. Old password must be validated before the update goes through. Standard hashing (already exists in the codebase).

Backend endpoints: profile GET/PUT, password-change PUT. All behind auth middleware.

## Acceptance criteria

- [x] Avatar picker shows a grid of PNG avatars, selecting one saves the filename to the user's profile
- [x] Display name and email are editable through the settings page
- [x] Password change requires old password, new password, and confirmation
- [x] All changes persist through page reload
- [x] Backend validates old password before accepting a password change

## Blocked by

None — can start immediately.
