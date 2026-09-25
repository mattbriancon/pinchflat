defmodule PinchflatWeb.CustomComponents.ButtonComponents do
  @moduledoc false
  use Phoenix.Component, global_prefixes: ~w(x-)

  alias PinchflatWeb.CoreComponents
  alias PinchflatWeb.CustomComponents.TextComponents

  @doc """
  Render a button

  ## Examples

      <.button color="bg-primary" rounding="rounded-sm">
        <span>Click me</span>
      </.button>
  """
  attr :color, :string, default: "bg-primary"
  attr :rounding, :string, default: "rounded-sm"
  attr :class, :string, default: ""
  attr :type, :string, default: "submit"
  attr :disabled, :boolean, default: false
  attr :rest, :global

  slot :inner_block, required: true

  def button(assigns) do
    ~H"""
    <button
      class={[
        "text-center font-medium text-white whitespace-nowrap",
        "#{@rounding} inline-flex items-center justify-center gap-1 px-4 py-2 text-sm",
        "#{@color}",
        "hover:bg-opacity-90",
        "disabled:bg-opacity-50 disabled:cursor-not-allowed disabled:text-grey-5",
        @class
      ]}
      type={@type}
      disabled={@disabled}
      {@rest}
    >
      {render_slot(@inner_block)}
    </button>
    """
  end

  @doc """
  Render a dropdown based off a button

  ## Examples

      <.button_dropdown text="Actions">
        <:option>TEST</:option>
      </.button_dropdown>
  """
  attr :text, :string, required: true
  attr :class, :string, default: ""

  slot :option, required: true

  def button_dropdown(assigns) do
    ~H"""
    <div x-data="{ dropdownOpen: false }" class={["relative flex", @class]}>
      <span
        x-on:click.prevent="dropdownOpen = !dropdownOpen"
        class={[
          "cursor-pointer inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm whitespace-nowrap",
          "font-medium text-white hover:bg-opacity-95"
        ]}
      >
        {@text}
        <CoreComponents.icon
          name="hero-chevron-down"
          class="fill-current duration-200 ease-linear h-4 w-4"
          x-bind:class="dropdownOpen && 'rotate-180'"
        />
      </span>
      <div
        x-show="dropdownOpen"
        x-on:click.outside="dropdownOpen = false"
        class="absolute right-0 top-full z-40 mt-1 min-w-full w-max rounded-lg border border-strokedark bg-graydark py-1 shadow-lg"
      >
        <ul class="flex flex-col">
          <li :for={option <- @option}>
            <span class="flex px-4 py-2 text-sm font-medium text-bodydark hover:bg-meta-4 hover:text-white cursor-pointer">
              {render_slot(option)}
            </span>
          </li>
        </ul>
      </div>
    </div>
    """
  end

  @doc """
  Render a button with an icon. Optionally include a tooltip.

  ## Examples

      <.icon_button icon_name="hero-check" tooltip="Complete" />
  """
  attr :icon_name, :string, required: true
  attr :class, :string, default: ""
  attr :tooltip, :string, default: nil
  attr :rest, :global

  def icon_button(assigns) do
    ~H"""
    <TextComponents.tooltip position="bottom" tooltip={@tooltip} tooltip_class="text-nowrap">
      <button
        class={[
          "flex justify-center items-center rounded-lg ",
          "bg-meta-4 border border-form-strokedark",
          "hover:bg-strokedark hover:text-white",
          @class
        ]}
        type="button"
        {@rest}
      >
        <CoreComponents.icon name={@icon_name} class="h-4 w-4" />
      </button>
    </TextComponents.tooltip>
    """
  end
end
