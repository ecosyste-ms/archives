Rails.application.routes.draw do
  mount Rswag::Ui::Engine => '/docs'
  mount Rswag::Api::Engine => '/docs'
  
  namespace :api, :defaults => {:format => :json} do
    namespace :v1 do
      resources :archives, only: [] do
        collection do
          get :list
          get :contents
          get :readme
        end
      end
    end
  end

  get '/404', to: 'errors#not_found'
  get '/422', to: 'errors#unprocessable'
  get '/500', to: 'errors#internal'

  root "home#index"
end  
