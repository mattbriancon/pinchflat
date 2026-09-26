defmodule PinchflatWeb.CustomComponents.TabComponents do
  @moduledoc false
  use Phoenix.Component

  @doc """
  Takes a list of tabs and renders them in a tabbed layout.
  """
  slot :tab, required: true do
    attr :id, :string, required: true
    attr :title, :string, required: true
  end

  slot :tab_append, required: false

  def tabbed_layout(assigns) do
    assigns = Map.put(assigns, :first_tab_id, hd(assigns.tab).id)

    ~H"""
    <div
      x-data={"{
        openTab: getTabFromHash('#{@first_tab_id}', '#{@first_tab_id}'),
        activeClasses: 'text-white border-meta-5',
        inactiveClasses: 'border-transparent'
      }"}
      @hashchange.window={"openTab = getTabFromHash(openTab, '#{@first_tab_id}')"}
      class="w-full"
    >
      <header class="flex flex-col md:flex-row md:items-center md:justify-between gap-2 border-b border-strokedark">
        <div class="no-scrollbar -mb-px flex gap-5 overflow-x-auto sm:gap-8">
          <a
            :for={tab <- @tab}
            href="#"
            @click.prevent={"openTab = setTabByName('#{tab.id}')"}
            x-bind:class={"openTab === '#{tab.id}' ? activeClasses : inactiveClasses"}
            class="border-b-2 py-2.5 whitespace-nowrap font-medium hover:text-meta-5"
          >
            {tab.title}
          </a>
        </div>
        <div class="mb-2 md:mb-0 flex gap-3 items-center empty:hidden">
          {render_slot(@tab_append)}
        </div>
      </header>
      <div class="mt-3 min-h-40 overflow-x-auto">
        <div :for={tab <- @tab} x-show={"openTab === '#{tab.id}'"} class="leading-normal">
          {render_slot(tab)}
        </div>
      </div>
    </div>
    """
  end
end
