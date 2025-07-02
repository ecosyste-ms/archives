module ApplicationHelper
  def meta_title
    [@meta_title, 'Ecosyste.ms: Archives'].compact.join(' | ')
  end

  def meta_description
    @meta_description || app_description
  end

  def app_name
    "Archives"
  end

  def app_description
    'An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.'
  end
end
