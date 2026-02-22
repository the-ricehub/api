-- required extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- table schemas
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username CITEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password TEXT NOT NULL,
    avatar_path TEXT,
    is_admin BOOL NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(author_id, slug)
);

CREATE TABLE rice_dotfiles (
    rice_id UUID PRIMARY KEY REFERENCES rices(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL UNIQUE,
    download_count INTEGER NOT NULL DEFAULT 0 CHECK (download_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_stars (
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    starred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(rice_id, user_id)
);

CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    rice_id UUID REFERENCES rices(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES rice_comments(id) ON DELETE CASCADE,
    is_closed BOOL NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- make sure that at least one object is referenced
    CHECK (
        (rice_id IS NOT NULL)::int + (comment_id IS NOT NULL)::int = 1
    ),
    -- create unique key to ensure users dont send duplicated reports
    UNIQUE(reporter_id, reason, is_closed)
);

-- junction tables
CREATE TABLE rices_tags (
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE
);

-- logic behind updating the `updated_at` column for all tables
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql'; 

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_tags_updated_at
    BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rices_updated_at
    BEFORE UPDATE ON rices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rice_dotfiles_updated_at
    BEFORE UPDATE ON rice_dotfiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rice_previews_updated_at
    BEFORE UPDATE ON rice_previews
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rice_comments_updated_at
    BEFORE UPDATE ON rice_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- insert test data
INSERT INTO tags (name)
VALUES ('AwesomeWM'), ('Arch Linux'), ('KDE'), ('Hyprland'), ('i3'), ('bspwm');

-- add dotfiles size column
ALTER TABLE rice_dotfiles
ADD COLUMN file_size BIGINT NOT NULL CHECK (file_size > 0);

-- tables related to website
CREATE TABLE website_variables (
    key TEXT PRIMARY KEY CHECK (key ~ '^[a-z0-9_]+$'),
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER update_website_variables_updated_at
    BEFORE UPDATE ON website_variables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE links (
    name TEXT PRIMARY KEY CHECK (name ~ '^[a-z]+$'),
    url TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER update_links_updated_at
    BEFORE UPDATE ON links
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- default values
INSERT INTO website_variables (key, value)
VALUES
    ('terms_of_service_text', 'Lorem ipsum'),
    ('privacy_policy_text', 'Lorem ipsum');

INSERT INTO links (name, url)
VALUES
    ('discord', 'https://discord.com'),
    ('github', 'https://github.com');