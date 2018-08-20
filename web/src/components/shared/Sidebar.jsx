import * as React from "react";
import autoBind from "react-autobind";
import assign from "object-assign";
import SidebarItem from "./SidebarItem";

// CSS
import "../../scss/components/shared/Sidebar.scss";

export default class Sidebar extends React.Component {
  constructor() {
    super();
    autoBind(this);
    this.state = {
      activeSub: "",
    }
  }

  isActive(link, pathname = "") {
    if (!link) return false;
    return pathname.indexOf(`${link}`) > -1;
  }

  buildSubItems(items) {
    const { activeSub } = this.state;
    const _items = items.map((item) => {
      return assign(item, {
        isActive: activeSub === item.id,
      });
    });
    return _items;
  }

  getSidebarItems(configOnly) {
    if (configOnly) {
      const { configRouteId } = this.props;
      return [{
        id: 0,
        linkTo: `/${configRouteId}`,
        label: "Application settings",
        position: "top",
        subItems: this.buildSubItems(this.props.appSettingsFieldsList)
      },
      ];
    } else {
      return [{
        id: 0,
        linkTo: "/config",
        label: "Application settings",
        position: "top",
        subItems: this.buildSubItems(this.props.appSettingsFieldsList)
      }, {
        id: 1,
        linkTo: `/support`,
        label: "Support",
        position: "top",
      }, {
        id: 2,
        linkTo: `/cluster`,
        label: "Cluster",
        position: "top",
      }, {
        id: 4,
        linkTo: `/releases`,
        label: "Releases",
        position: "top",
      }, {
        id: 5,
        linkTo: `/snapshots`,
        label: "Snapshots",
        position: "top",
      }, {
        id: 6,
        linkTo: `/view-license`,
        label: "View license",
        position: "top",
      }, {
        id: 7,
        linkTo: `/console-settings`,
        label: "Console settings",
        position: "top",
        subItems: this.buildSubItems(this.props.consoleSettingsFieldsList)
      },
      ];
    }
  }

  scrollToSection(id) {
    const el = document.getElementById(id);
    if (!el) return;
    setTimeout(() => {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
      this.setState({ activeSub: id });
    }, 50);
  }

  componentDidUpdate(lastProps) {
    const { location } = this.props;
    if (lastProps.location.hash !== location.hash && location.hash) {
      this.scrollToSection(location.hash.replace("#", ""));
    }
  }

  onClick(item) {
    this.setState({ route: item.linkTo });
    if (item.subItems) {
      this.setState({ activeSub: "" });
    }
    return (e, ...rest) => {
      if (typeof item.onClick === "function") {
        item.onClick(e, ...rest);
        return;
      }
    };
  }

  render() {
    const { configOnly } = this.props;
    const items = this.getSidebarItems(configOnly);
    const sidebarItems = items.map(item => {
      const active = this.isActive(item.linkTo, window.location.pathname);
      return assign(item, {
        isActive: active,
      });
    });
    const renderItem = item => {
      return (
        <div key={item.id}>
          <SidebarItem
            {...item}
            onClick={() => this.onClick(item)}
          />
          {item.isActive && item.subItems ?
            item.subItems.map((item) => (
              <SidebarItem
                key={item.id}
                className="SubItem"
                activeSub={this.state.activeSub}
                subItemLinkTo={`${this.props.location.pathname}#${item.id}`}
                {...item}
                onClick={() => this.scrollToSection(item.id)}
              />
            ))
            : null}
        </div>
      );
    };

    return (
      <div className="SidebarContent-wrapper flex flex1 u-minHeight--full">
        <div className="flex-column u-width--full">
          <div className="SidebarElements-wrapper flex-column flex-1-auto u-overflow--auto">
            {sidebarItems
              .filter(i => i.position === "top")
              .map(renderItem)
            }
          </div>
        </div>
      </div>
    );
  }
}