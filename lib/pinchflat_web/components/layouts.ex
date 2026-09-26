defmodule PinchflatWeb.Layouts do
  use PinchflatWeb, :html

  embed_templates "layouts/*"
  embed_templates "layouts/partials/*"

  @doc """
  Renders a link in the top navigation bar, highlighted when it matches the current page

  ## Examples

      <.nav_link text="Sources" href="/sources" conn={@conn} />
  """
  attr :text, :string, required: true
  attr :href, :any, required: true
  attr :conn, Plug.Conn, required: true

  def nav_link(assigns) do
    assigns = assign(assigns, :current_path, Phoenix.Controller.current_path(assigns.conn, %{}))

    ~H"""
    <.link
      href={@href}
      class={[
        "shrink-0 whitespace-nowrap rounded-lg px-2 py-1.5 text-sm font-medium sm:px-2.5",
        "hover:bg-meta-4 hover:text-white",
        if(active_path?(@href, @current_path), do: "bg-meta-4 text-white", else: "text-bodydark1")
      ]}
    >
      {@text}
    </.link>
    """
  end

  @doc """
  Renders a link in the page footer

  ## Examples

      <.footer_link icon="si-github" text="Github" href="https://github.com" />
  """
  attr :icon, :string, required: true
  attr :text, :string, required: true
  attr :href, :string, required: true

  def footer_link(assigns) do
    ~H"""
    <.link href={@href} target="_blank" class="flex items-center gap-1.5 hover:text-white">
      <.icon name={@icon} class="h-4 w-4" /> {@text}
    </.link>
    """
  end

  defp active_path?(_href, nil), do: false
  defp active_path?(href, current_path) when href == current_path, do: true
  defp active_path?("/", _current_path), do: false
  defp active_path?(href, current_path), do: String.starts_with?(current_path, href <> "/")
end
