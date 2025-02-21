FROM ruby:3.4.2-alpine

ENV APP_ROOT=/usr/src/app
ENV DATABASE_PORT=5432
ENV PIP_BREAK_SYSTEM_PACKAGES=1
WORKDIR $APP_ROOT

COPY Gemfile Gemfile.lock $APP_ROOT/

RUN apk add --no-cache \
    build-base \
    netcat-openbsd \
    git \
    tzdata \
    curl-dev \
    libc6-compat \
    tar \
    libarchive-tools \
    icu-dev \
    cmake \
    perl \
    libidn-dev \
    py-pip \
    nodejs \
    npm \
    yaml-dev \
    libffi-dev \
 && gem update --system \
 && gem install bundler foreman \
 && bundle config set without 'test development' \
 && bundle install --jobs 8 \
 && pip install docutils \
 && npm install -g repomix

COPY . $APP_ROOT

RUN RAILS_ENV=production bundle exec rake assets:precompile

CMD ["bin/docker-start"]