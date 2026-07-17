import { Text, VStack } from "@expo/ui/swift-ui";
import { font, foregroundStyle, widgetURL } from "@expo/ui/swift-ui/modifiers";
import { createWidget, type WidgetEnvironment } from "expo-widgets";

export interface MyTasksWidgetProps {
  count: number;
  nextTaskId: string;
  nextTaskTitle: string;
  spaceSlug: string;
}

const MyTasksWidget = (props: MyTasksWidgetProps, environment: WidgetEnvironment) => {
  "widget";

  return (
    <VStack
      spacing={6}
      modifiers={[widgetURL(`horologia://task/${props.spaceSlug}/${props.nextTaskId}`)]}
    >
      <Text modifiers={[font({ weight: "bold", size: 17 })]}>My Tasks</Text>
      <Text modifiers={[font({ weight: "bold", size: 32 }), foregroundStyle("#2F6D4B")]}>
        {props.count}
      </Text>
      <Text>{props.nextTaskTitle}</Text>
      <Text modifiers={[font({ size: 11 }), foregroundStyle("#66736B")]}>
        {environment.widgetFamily}
      </Text>
    </VStack>
  );
};

export default createWidget("MyTasksWidget", MyTasksWidget);
