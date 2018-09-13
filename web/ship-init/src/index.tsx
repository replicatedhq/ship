// TODO: Remove when TypeScript used in Ship
// @ts-ignore
import { Ship as ShipComponent } from "./Ship";

export interface ShipInitProps {
  apiEndpoint: string;
}

export const Ship: React.Component<ShipInitProps> = ShipComponent;

