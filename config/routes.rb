Rails.application.routes.draw do
  Rails.application.routes.draw do
    mount Rswag::Ui::Engine => '/docs'
    mount Rswag::Api::Engine => '/docs'
    
    namespace :api, :defaults => {:format => :json} do
      namespace :v1 do
        resources :archives, only: [] do
          collection do
            get :list
            get :contents
          end
        end
      end
    end
  
    root "home#index"
  end  
end
