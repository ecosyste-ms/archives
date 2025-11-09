#!/usr/bin/env -S falcon host
# frozen_string_literal: true

load :rack

hostname = File.basename(__dir__)
port = ENV.fetch('PORT', 5000).to_i

rack hostname do
  endpoint Async::HTTP::Endpoint.parse("http://0.0.0.0:#{port}")
end
