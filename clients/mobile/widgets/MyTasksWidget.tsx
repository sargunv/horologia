import { Text, VStack } from "@expo/ui/swift-ui";
import { font, foregroundStyle, widgetURL } from "@expo/ui/swift-ui/modifiers";
import { createWidget, type WidgetEnvironment } from "expo-widgets";

export interface MyTasksWidgetProps {
  count: number;
  generatedAt: string;
  nextTaskId: string;
  nextTaskTitle: string;
  secondTaskTitle: string;
  signedIn: boolean;
  spaceSlug: string;
  thirdTaskTitle: string;
}

const MyTasksWidget = (props: MyTasksWidgetProps, environment: WidgetEnvironment) => {
  "widget";

  const isSmall = environment.widgetFamily === "systemSmall";
  const destination = props.nextTaskId
    ? `horologia://task/${props.spaceSlug}/${props.nextTaskId}`
    : "horologia://";
  return (
    <VStack spacing={6} modifiers={[widgetURL(destination)]}>
      <Text modifiers={[font({ weight: "bold", size: 17 })]}>My Tasks</Text>
      <Text modifiers={[font({ weight: "bold", size: 32 }), foregroundStyle("#2F6D4B")]}>
        {props.count}
      </Text>
      <Text>{props.nextTaskTitle}</Text>
      {!isSmall && props.secondTaskTitle ? <Text>{props.secondTaskTitle}</Text> : null}
      {!isSmall && props.thirdTaskTitle ? <Text>{props.thirdTaskTitle}</Text> : null}
      <Text modifiers={[font({ size: 11 }), foregroundStyle("#66736B")]}>
        {props.signedIn ? `Saved ${props.generatedAt}` : "Open Horologia to sign in"}
      </Text>
    </VStack>
  );
};

export default createWidget("MyTasksWidget", MyTasksWidget);
